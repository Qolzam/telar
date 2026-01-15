package moderation

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"time"

	"github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/pretrained"
	ort "github.com/yalue/onnxruntime_go"
)

// ONNXConfig holds configuration for an ONNX model
type ONNXConfig struct {
	ModelPath     string
	Name          string  // e.g., "toxicity", "spam"
	Threshold     float64 // Threshold for flagging (e.g., 0.90 for toxicity, 0.80 for spam)
	BadClassIndex int     // Index of the "bad" class in model output (typically 1 for [safe, bad])
}

type ONNXModerator struct {
	session   *ort.DynamicAdvancedSession
	tokenizer *tokenizer.Tokenizer
	config    ONNXConfig
}

func NewONNXModerator(modelPath string, config ONNXConfig) (*ONNXModerator, error) {
	// ONNX Runtime must be initialized globally before calling this
	// Load model session only
	inputNames := []string{"input_ids", "attention_mask"}
	outputNames := []string{"logits"}
	session, err := ort.NewDynamicAdvancedSession(modelPath, inputNames, outputNames, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Load Tokenizer from local files (production-ready, no network calls)
	// Try to load from local tokenizer.json first, fallback to pretrained if not found
	modelDir := filepath.Dir(modelPath)
	tokenizerPath := filepath.Join(modelDir, "tokenizer.json")

	var tk *tokenizer.Tokenizer
	var loadErr error
	if tk, loadErr = pretrained.FromFile(tokenizerPath); loadErr != nil {
		// Fallback to pretrained (for development/testing)
		// This will work in development but may fail in Docker if Go module cache not available
		tk = pretrained.BertBaseUncased()
	}

	// Set default bad class index if not specified
	if config.BadClassIndex == 0 {
		config.BadClassIndex = 1 // Default: index 1 is the "bad" class
	}

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

	// 1. Tokenize
	en, err := m.tokenizer.EncodeSingle(text)
	if err != nil {
		return nil, err
	}

	// 2. Prepare Tensors
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
	outputTensor, _ := ort.NewEmptyTensor[float32](ort.NewShape(1, 2))

	// 3. Inference
	err = m.session.Run(
		[]ort.ArbitraryTensor{inputTensor, maskTensor},
		[]ort.ArbitraryTensor{outputTensor},
	)
	if err != nil {
		return nil, err
	}

	// 4. Process Output (Softmax)
	logits := outputTensor.GetData()
	probs := softmax(logits)

	// Get probability of the "bad" class (e.g., toxic, spam)
	if m.config.BadClassIndex >= len(probs) {
		return nil, fmt.Errorf("bad class index %d out of range (model outputs %d classes)", m.config.BadClassIndex, len(probs))
	}
	badProb := probs[m.config.BadClassIndex]

	// 5. Decision Logic: Only return a result if we detect a violation
	// If the score is below threshold, return nil to pass through to next check.
	if float64(badProb) > m.config.Threshold {
		// Flagged: Return result with violation detected
		scoreKey := m.config.Name // e.g., "toxicity" or "spam"
		return &ModerationResult{
			IsFlagged:       true,
			FlagReason:      fmt.Sprintf("onnx_%s_high", m.config.Name),
			SuggestedAction: "review_needed",
			Scores:          map[string]float64{scoreKey: float64(badProb), "confidence": float64(badProb)},
			AnalysisTimeMs:  time.Since(start).Milliseconds(),
			ModelUsed:       m.Name(),
		}, nil
	}

	// Not flagged by this model: Return nil to pass through to next check
	// This allows the pipeline to continue checking with other models (e.g., spam after toxicity)
	return nil, nil
}

// Helper softmax (same as POC)
func softmax(logits []float32) []float32 {
	if len(logits) == 0 {
		return nil
	}

	// Find max for numerical stability
	max := logits[0]
	for _, v := range logits {
		if v > max {
			max = v
		}
	}

	// Compute exp(x - max) for each element
	exps := make([]float32, len(logits))
	sum := float32(0.0)
	for i, v := range logits {
		exps[i] = float32(math.Exp(float64(v - max)))
		sum += exps[i]
	}

	// Normalize
	probs := make([]float32, len(logits))
	for i, exp := range exps {
		probs[i] = exp / sum
	}

	return probs
}
