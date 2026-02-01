"""LLM providers for email classification."""

from .base import (
    BatchClassificationResult,
    ClassificationResult,
    LLMProvider,
    ValidationResult,
    get_provider,
)

__all__ = [
    "LLMProvider",
    "ClassificationResult",
    "ValidationResult",
    "BatchClassificationResult",
    "get_provider",
]
