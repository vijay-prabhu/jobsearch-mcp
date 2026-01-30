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
