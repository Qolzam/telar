#!/usr/bin/env python3
"""
Export Models to ONNX Format
Supports multiple model types:
  - toxicity: martin-ha/toxic-comment-model (DistilBERT trained on Jigsaw toxicity dataset)
  - toxic-bert: unitary/toxic-bert (BERT multi-label classifier for Jigsaw Toxic Comment Taxonomy)
  - spam: mshenoda/roberta-spam (RoBERTa trained for spam detection)

Usage:
  python export_model.py toxicity [--output-dir /path/to/output]
  python export_model.py toxic-bert [--output-dir /path/to/output]
  python export_model.py spam [--output-dir /path/to/output]
"""
from transformers import AutoTokenizer, AutoModelForSequenceClassification
import torch
import os
import sys
import argparse

# Model configurations
MODEL_CONFIGS = {
    "toxicity": {
        "model_name": "martin-ha/toxic-comment-model",
        "default_output_dir": "/app/models/distilbert",
        "description": "DistilBERT toxicity classifier"
    },
    "toxic-bert": {
        "model_name": "unitary/toxic-bert",
        "default_output_dir": "/app/models/toxic-bert",
        "description": "BERT multi-label classifier for Jigsaw Toxic Comment Taxonomy"
    },
    "spam": {
        "model_name": "mshenoda/roberta-spam",
        "default_output_dir": "/app/models/spam",
        "description": "RoBERTa spam classifier"
    }
}

def export_model(model_type: str, output_dir: str = None):
    """Export a model to ONNX format"""
    if model_type not in MODEL_CONFIGS:
        print(f"❌ Error: Unknown model type '{model_type}'")
        print(f"Available types: {', '.join(MODEL_CONFIGS.keys())}")
        sys.exit(1)

    config = MODEL_CONFIGS[model_type]
    model_name = config["model_name"]
    
    if output_dir is None:
        output_dir = os.getenv("MODEL_OUTPUT_DIR", config["default_output_dir"])

    print(f"[EXPORT] Model Type: {model_type} ({config['description']})")
    print(f"[EXPORT] Loading model: {model_name}")
    print(f"[EXPORT] Output directory: {output_dir}")

    # Check if models already exist (skip if SKIP_IF_EXISTS)
    if os.getenv("SKIP_IF_EXISTS", "false").lower() == "true":
        required_files = ["model.onnx", "tokenizer.json"]
        if all(os.path.exists(os.path.join(output_dir, f)) for f in required_files):
            print(f"[EXPORT] Models already exist in {output_dir}, skipping export")
            return

    if not os.path.exists(output_dir):
        os.makedirs(output_dir)

    # 1. Download Tokenizer
    print("[EXPORT] Downloading tokenizer...")
    tokenizer = AutoTokenizer.from_pretrained(model_name)
    tokenizer.save_pretrained(output_dir)

    # 2. Download Model
    print("[EXPORT] Downloading model...")
    model = AutoModelForSequenceClassification.from_pretrained(model_name)
    model.eval()

    # 3. Export to ONNX
    print("[EXPORT] Exporting to ONNX...")
    dummy_input = tokenizer("This is a sample", return_tensors="pt")

    # For multi-label models (toxic-bert), output shape should be fixed [batch_size, 6]
    # For binary models, output shape should be fixed [batch_size, 2]
    # Only make input dimensions dynamic to handle variable sequence lengths
    dynamic_axes_config = {
        "input_ids": {0: "batch_size", 1: "sequence_length"},
        "attention_mask": {0: "batch_size", 1: "sequence_length"},
    }
    
    # Only add dynamic output axis for binary models (2 outputs)
    # Multi-label models (6 outputs) should have fixed output shape
    if model_type == "toxic-bert":
        # Fixed output shape [batch_size, 6] - don't make it dynamic
        pass  # Don't add logits to dynamic_axes
    else:
        # Binary models can have dynamic batch dimension
        dynamic_axes_config["logits"] = {0: "batch_size"}

    torch.onnx.export(
        model,
        (dummy_input["input_ids"], dummy_input["attention_mask"]),
        f"{output_dir}/model.onnx",
        input_names=["input_ids", "attention_mask"],
        output_names=["logits"],
        dynamic_axes=dynamic_axes_config,
        opset_version=18
    )

    print(f"✅ {model_type.capitalize()} Model Exported!")
    print(f"[EXPORT] Model saved to: {output_dir}/model.onnx")
    print(f"[EXPORT] Tokenizer files saved to: {output_dir}/")


def main():
    parser = argparse.ArgumentParser(
        description="Export models to ONNX format",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python export_model.py toxicity
  python export_model.py spam --output-dir /app/models/spam
  python export_model.py toxicity spam  # Export both models
  python export_model.py all  # Export all models
        """
    )
    parser.add_argument(
        "model_types",
        nargs="*",  # Changed from "+" to "*" to allow no arguments (read from env)
        choices=list(MODEL_CONFIGS.keys()) + ["all"],
        help="Model type(s) to export (toxicity, spam, or 'all'). If not provided, reads from MODEL_TYPES env var or defaults to 'all'"
    )
    parser.add_argument(
        "--output-dir",
        help="Output directory (overrides default and MODEL_OUTPUT_DIR env var)"
    )

    args = parser.parse_args()

    # Get model types from args, env var, or default to "all"
    if args.model_types:
        model_types_arg = args.model_types
    else:
        # Read from environment variable or default to "all"
        env_model_types = os.getenv("MODEL_TYPES", "all")
        model_types_arg = env_model_types.split()

    # Handle "all" option
    if "all" in model_types_arg:
        model_types = list(MODEL_CONFIGS.keys())
    else:
        model_types = model_types_arg

    # Export each model
    for model_type in model_types:
        if model_type == "all":
            continue
        export_model(model_type, args.output_dir)
        print()  # Blank line between exports


if __name__ == "__main__":
    main()

