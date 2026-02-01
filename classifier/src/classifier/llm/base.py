"""Base LLM provider interface and common types."""

from abc import ABC, abstractmethod
from typing import Optional

from pydantic import BaseModel


class ClassificationResult(BaseModel):
    """Result of email classification."""

    is_job_related: bool
    confidence: float
    company: Optional[str] = None
    position: Optional[str] = None
    recruiter_name: Optional[str] = None
    classification: Optional[str] = None
    reasoning: Optional[str] = None


class ValidationResult(BaseModel):
    """Result of structured validation for an email."""

    is_direct_opportunity: bool = False
    is_recruiter_outreach: bool = False
    is_interview_related: bool = False
    is_job_alert_newsletter: bool = False
    is_marketing_promo: bool = False
    is_application_response: bool = False
    final_verdict: bool = False
    confidence: float = 0.0
    reasoning: Optional[str] = None


class BatchClassificationResult(BaseModel):
    """Result of batch email classification."""

    results: list[ClassificationResult]
    batch_size: int = 0


class LLMProvider(ABC):
    """Abstract base class for LLM providers."""

    @abstractmethod
    async def classify(
        self,
        subject: str,
        body: str,
        from_address: str,
    ) -> ClassificationResult:
        """Classify an email and extract relevant information."""
        pass

    @abstractmethod
    async def validate(
        self,
        subject: str,
        body: str,
        from_address: str,
    ) -> ValidationResult:
        """Validate classification with structured multi-signal questions."""
        pass

    @abstractmethod
    async def classify_batch(
        self,
        emails: list[dict],
    ) -> BatchClassificationResult:
        """Classify multiple emails in a single LLM call."""
        pass

    @abstractmethod
    async def health_check(self) -> bool:
        """Check if the provider is available."""
        pass


def get_provider(name: str, **kwargs) -> LLMProvider:
    """Factory function to get an LLM provider by name."""
    if name == "ollama":
        from .ollama import OllamaProvider

        return OllamaProvider(**kwargs)
    elif name == "openai":
        from .openai import OpenAIProvider

        return OpenAIProvider(**kwargs)
    else:
        raise ValueError(f"Unknown provider: {name}")
