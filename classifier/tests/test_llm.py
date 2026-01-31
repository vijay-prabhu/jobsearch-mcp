"""Tests for LLM providers."""

import pytest

from classifier.llm.base import ClassificationResult, get_provider


class TestClassificationResult:
    """Tests for the ClassificationResult model."""

    def test_minimal_result(self):
        """Can create result with just required fields."""
        result = ClassificationResult(
            is_job_related=True,
            confidence=0.85,
        )
        assert result.is_job_related is True
        assert result.confidence == 0.85
        assert result.company is None
        assert result.position is None

    def test_full_result(self):
        """Can create result with all fields."""
        result = ClassificationResult(
            is_job_related=True,
            confidence=0.95,
            company="Stripe",
            position="Senior Engineer",
            recruiter_name="Jane Smith",
            classification="recruiter_outreach",
            reasoning="Direct outreach about specific role",
        )
        assert result.company == "Stripe"
        assert result.position == "Senior Engineer"
        assert result.recruiter_name == "Jane Smith"
        assert result.classification == "recruiter_outreach"

    def test_result_serialization(self):
        """Result should serialize to dict correctly."""
        result = ClassificationResult(
            is_job_related=True,
            confidence=0.9,
            company="Google",
        )
        data = result.model_dump()
        assert data["is_job_related"] is True
        assert data["confidence"] == 0.9
        assert data["company"] == "Google"


class TestProviderFactory:
    """Tests for the LLM provider factory."""

    def test_get_ollama_provider(self):
        """Factory should return OllamaProvider."""
        provider = get_provider("ollama")
        assert provider is not None
        assert provider.model == "llama3.2:1b"
        assert provider.host == "http://localhost:11434"

    def test_get_ollama_with_custom_model(self):
        """Factory should accept custom model for Ollama."""
        provider = get_provider("ollama", model="llama3:8b")
        assert provider.model == "llama3:8b"

    def test_get_openai_provider(self):
        """Factory should return OpenAIProvider."""
        provider = get_provider("openai")
        assert provider is not None

    def test_unknown_provider_raises(self):
        """Factory should raise for unknown provider."""
        with pytest.raises(ValueError, match="Unknown provider"):
            get_provider("invalid_provider")


class TestOllamaProvider:
    """Tests for OllamaProvider-specific functionality."""

    def test_default_initialization(self):
        """Provider initializes with defaults."""
        provider = get_provider("ollama")
        assert provider.model == "llama3.2:1b"
        assert provider.host == "http://localhost:11434"
        assert provider.timeout == 60.0

    def test_custom_host(self):
        """Provider accepts custom host."""
        provider = get_provider("ollama", host="http://custom:11434")
        assert provider.host == "http://custom:11434"

    def test_host_strips_trailing_slash(self):
        """Provider strips trailing slash from host."""
        provider = get_provider("ollama", host="http://localhost:11434/")
        assert provider.host == "http://localhost:11434"

    def test_parse_valid_json(self):
        """Provider parses valid JSON response."""
        provider = get_provider("ollama")
        content = """{
            "is_job_related": true,
            "confidence": 0.9,
            "company": "TestCorp",
            "classification": "recruiter_outreach"
        }"""
        result = provider._parse_response(content)
        assert result.is_job_related is True
        assert result.confidence == 0.9
        assert result.company == "TestCorp"

    def test_parse_json_with_extra_text(self):
        """Provider extracts JSON from text with extra content."""
        provider = get_provider("ollama")
        content = """Here is my analysis:
        {"is_job_related": true, "confidence": 0.8}
        Hope this helps!"""
        result = provider._parse_response(content)
        assert result.is_job_related is True
        assert result.confidence == 0.8

    def test_parse_invalid_json_returns_fallback(self):
        """Provider returns fallback for invalid JSON."""
        provider = get_provider("ollama")
        content = "This is not valid JSON"
        result = provider._parse_response(content)
        assert result.is_job_related is False
        assert result.confidence == 0.0
        assert "Parse error" in result.reasoning

    def test_fallback_result(self):
        """Fallback result is conservative."""
        provider = get_provider("ollama")
        result = provider._fallback_result("Test failure")
        assert result.is_job_related is False
        assert result.confidence == 0.0
        assert "Test failure" in result.reasoning
