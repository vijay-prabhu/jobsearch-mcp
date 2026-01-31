"""LLM providers for email classification."""

from .base import ClassificationResult, LLMProvider, get_provider

__all__ = ["LLMProvider", "ClassificationResult", "get_provider"]
