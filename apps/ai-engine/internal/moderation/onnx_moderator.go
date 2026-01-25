package moderation

import (
	"context"
	"fmt"
	"log"
	"math"
	"path/filepath"
	"time"

	"github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/pretrained"
	ort "github.com/yalue/onnxruntime_go"
)

// The Jigsaw Toxic Comment Taxonomy (for multi-label models)
var forensicLabels = []string{
	"toxic",
	"severe_toxic",
	"obscene",
	"threat",
	"insult",
	"identity_hate",
}

// ONNXConfig holds configuration for an ONNX model
type ONNXConfig struct {
	ModelPath     string
	Name          string  // e.g., "toxicity", "spam", "toxic-bert"
	Threshold     float64 // Threshold for binary classification flagging (e.g., 0.90 for toxicity, 0.80 for spam)
	BadClassIndex int     // Index of the "bad" class in binary model output (typically 1 for [safe, bad])
	// Thresholds map for multi-label classification (e.g., toxic-bert with 6 outputs)
	// If set, the model is treated as multi-label and Threshold/BadClassIndex are ignored
	Thresholds map[string]float64 `json:"thresholds,omitempty"`
}

type ONNXModerator struct {
	session   *ort.DynamicAdvancedSession
	tokenizer *tokenizer.Tokenizer
	config    ONNXConfig
}

func NewONNXModerator(modelPath string, config ONNXConfig) (*ONNXModerator, error) {
	// ONNX Runtime must be initialized globally before calling this
	inputNames := []string{"input_ids", "attention_mask"}
	outputNames := []string{"logits"}
	session, err := ort.NewDynamicAdvancedSession(modelPath, inputNames, outputNames, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Load tokenizer from local files (production) or fallback to pretrained (dev)
	modelDir := filepath.Dir(modelPath)
	tokenizerPath := filepath.Join(modelDir, "tokenizer.json")

	var tk *tokenizer.Tokenizer
	var loadErr error
	if tk, loadErr = pretrained.FromFile(tokenizerPath); loadErr != nil {
		tk = pretrained.BertBaseUncased()
	}

	if config.BadClassIndex == 0 {
		config.BadClassIndex = 1
	}

	// Startup validation: verify model output dimensions match expected type
	isMultiLabel := len(config.Thresholds) > 0
	expectedDims := 2 // Default to binary
	if isMultiLabel {
		expectedDims = len(forensicLabels) // Should be 6 for toxic-bert
	}

	dummyText := "startup_validation_check"
	en, err := tk.EncodeSingle(dummyText, true)
	if err != nil {
		return nil, fmt.Errorf("startup validation failed (tokenizer): %w", err)
	}
	inputIDs := make([]int64, len(en.Ids))
	attentionMask := make([]int64, len(en.AttentionMask))
	for i, v := range en.Ids {
		inputIDs[i] = int64(v)
	}
	for i, v := range en.AttentionMask {
		attentionMask[i] = int64(v)
	}

	inputShape := []int64{1, int64(len(inputIDs))}
	inputTensor, err := ort.NewTensor(ort.NewShape(inputShape...), inputIDs)
	if err != nil {
		return nil, fmt.Errorf("startup validation failed (input tensor): %w", err)
	}
	maskTensor, err := ort.NewTensor(ort.NewShape(inputShape...), attentionMask)
	if err != nil {
		return nil, fmt.Errorf("startup validation failed (mask tensor): %w", err)
	}

	outputTensorFloat32, err := ort.NewEmptyTensor[float32](ort.NewShape(inputShape[0], int64(expectedDims)))
	if err != nil {
		return nil, fmt.Errorf("startup validation failed (output tensor allocation): %w", err)
	}
	err = session.Run(
		[]ort.ArbitraryTensor{inputTensor, maskTensor},
		[]ort.ArbitraryTensor{outputTensorFloat32},
	)
	if err != nil {
		return nil, fmt.Errorf("startup validation failed (inference): %w", err)
	}

	outputData := outputTensorFloat32.GetData()
	actualDims := len(outputData)

	if actualDims != expectedDims {
		modelType := "binary"
		if isMultiLabel {
			modelType = "multi-label (toxic-bert)"
		}
		return nil, fmt.Errorf(
			"CRITICAL MODEL MISMATCH: Code expects %s model (%d outputs), but loaded ONNX model has %d outputs. "+
				"Check MODERATION_ONNX_TOXICITY_MODEL_PATH in your .env file. "+
				"Expected: %s model at %s, Got: model with %d outputs",
			modelType, expectedDims, actualDims,
			modelType, modelPath, actualDims,
		)
	}

	log.Printf("✅ ONNX Model Verified: %s model loaded successfully (output dimension %d matches expected %d)", config.Name, actualDims, expectedDims)

	return &ONNXModerator{
		session:   session,
		tokenizer: tk,
		config:    config,
	}, nil
}

func (m *ONNXModerator) Name() string {
	return "L3-ONNX-" + m.config.Name
}

func (m *ONNXModerator) Moderate(ctx context.Context, text string) (*ModerationResult, error) {
	start := time.Now()

	isMultiLabel := len(m.config.Thresholds) > 0

	en, err := m.tokenizer.EncodeSingle(text, true)
	if err != nil {
		return nil, err
	}
	inputIDs := make([]int64, len(en.Ids))
	attentionMask := make([]int64, len(en.AttentionMask))
	for i, v := range en.Ids {
		inputIDs[i] = int64(v)
	}
	for i, v := range en.AttentionMask {
		attentionMask[i] = int64(v)
	}

	inputShape := []int64{1, int64(len(inputIDs))}
	inputTensor, _ := ort.NewTensor(ort.NewShape(inputShape...), inputIDs)
	maskTensor, _ := ort.NewTensor(ort.NewShape(inputShape...), attentionMask)

	var outputDim int64
	if isMultiLabel {
		outputDim = 6
	} else {
		outputDim = 2
	}

	outputTensorFloat32, err := ort.NewEmptyTensor[float32](ort.NewShape(inputShape[0], outputDim))
	if err != nil {
		return nil, fmt.Errorf("failed to allocate output tensor: %w", err)
	}
	err = m.session.Run(
		[]ort.ArbitraryTensor{inputTensor, maskTensor},
		[]ort.ArbitraryTensor{outputTensorFloat32},
	)
	if err != nil {
		return nil, err
	}

	logits := outputTensorFloat32.GetData()

	if isMultiLabel {
		if len(logits) != 6 {
			return nil, fmt.Errorf("expected 6 logits for multi-label model, got %d", len(logits))
		}
		return m.processMultiLabelOutput(logits, start)
	} else {
		if len(logits) != 2 {
			return nil, fmt.Errorf("expected 2 logits for binary model, got %d", len(logits))
		}
		return m.processBinaryOutput(logits, start)
	}
}

// processMultiLabelOutput handles multi-label classification (6 outputs with sigmoid)
func (m *ONNXModerator) processMultiLabelOutput(logits []float32, start time.Time) (*ModerationResult, error) {
	if len(logits) != 6 {
		err := fmt.Errorf("expected 6 logits for multi-label model, got %d", len(logits))
		return nil, err
	}

	scores := make(map[string]float64)
	for i, label := range forensicLabels {
		scores[label] = float64(sigmoid(logits[i]))
	}

	// Decision matrix: check specific violations first, generic toxicity last
	violation := ""
	thresh := m.config.Thresholds

	if scores["threat"] >= thresh["threat"] {
		violation = "Violence/Threat"
	} else if scores["identity_hate"] >= thresh["identity_hate"] {
		violation = "Hate Speech"
	} else if scores["severe_toxic"] >= thresh["severe_toxic"] {
		violation = "Severe Toxicity"
	} else if scores["insult"] >= thresh["insult"] {
		violation = "Personal Insult"
	} else if scores["obscene"] >= thresh["obscene"] {
		violation = "Obscene Content"
	} else if scores["toxic"] >= thresh["toxic"] {
		violation = "High Toxicity"
	}

	if violation != "" {
		result := &ModerationResult{
			IsFlagged:       true,
			FlagReason:      "onnx_" + violation,
			Scores:          scores, // Pass all 6 scores to API
			SuggestedAction: "review_needed",
			AnalysisTimeMs:  time.Since(start).Milliseconds(),
			ModelUsed:       m.Name(),
		}

		return result, nil
	}

	return nil, nil
}

// processBinaryOutput handles binary classification (2 outputs with softmax)
func (m *ONNXModerator) processBinaryOutput(logits []float32, start time.Time) (*ModerationResult, error) {
	if len(logits) != 2 {
		err := fmt.Errorf("expected 2 logits for binary model, got %d", len(logits))
		return nil, err
	}

	probs := softmax(logits)

	if m.config.BadClassIndex >= len(probs) {
		err := fmt.Errorf("bad class index %d out of range (model outputs %d classes)", m.config.BadClassIndex, len(probs))
		return nil, err
	}
	badProb := probs[m.config.BadClassIndex]
	badProbFloat := float64(badProb)
	if badProbFloat > m.config.Threshold {
		scoreKey := m.config.Name
		result := &ModerationResult{
			IsFlagged:       true,
			FlagReason:      fmt.Sprintf("onnx_%s_high", m.config.Name),
			SuggestedAction: "review_needed",
			Scores:          map[string]float64{scoreKey: badProbFloat, "confidence": badProbFloat},
			AnalysisTimeMs:  time.Since(start).Milliseconds(),
			ModelUsed:       m.Name(),
		}
		return result, nil
	}

	return nil, nil
}

func softmax(logits []float32) []float32 {
	if len(logits) == 0 {
		return nil
	}

	max := logits[0]
	for _, v := range logits {
		if v > max {
			max = v
		}
	}

	exps := make([]float32, len(logits))
	sum := float32(0.0)
	for i, v := range logits {
		exps[i] = float32(math.Exp(float64(v - max)))
		sum += exps[i]
	}

	probs := make([]float32, len(logits))
	for i, exp := range exps {
		probs[i] = exp / sum
	}

	return probs
}

func sigmoid(x float32) float32 {
	return 1.0 / (1.0 + float32(math.Exp(float64(-x))))
}
