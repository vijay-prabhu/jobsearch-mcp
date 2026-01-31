"""Tests for prompt templates."""

from classifier.prompts import CLASSIFICATION_PROMPT, EXTRACTION_PROMPT


class TestClassificationPrompt:
    """Tests for the classification prompt template."""

    def test_prompt_has_placeholders(self):
        """Prompt should contain required placeholders."""
        assert "{subject}" in CLASSIFICATION_PROMPT
        assert "{body}" in CLASSIFICATION_PROMPT
        assert "{from_address}" in CLASSIFICATION_PROMPT

    def test_prompt_format(self):
        """Prompt should format correctly."""
        formatted = CLASSIFICATION_PROMPT.format(
            subject="Test Subject",
            body="Test Body",
            from_address="test@example.com",
        )
        assert "Test Subject" in formatted
        assert "Test Body" in formatted
        assert "test@example.com" in formatted

    def test_prompt_includes_guidelines(self):
        """Prompt should include classification guidelines."""
        assert "recruiter_outreach" in CLASSIFICATION_PROMPT
        assert "application_confirmation" in CLASSIFICATION_PROMPT
        assert "interview_request" in CLASSIFICATION_PROMPT
        assert "rejection" in CLASSIFICATION_PROMPT
        assert "offer" in CLASSIFICATION_PROMPT

    def test_prompt_includes_exclusions(self):
        """Prompt should list types to exclude."""
        assert "Job alert digests" in CLASSIFICATION_PROMPT
        assert "LinkedIn notifications" in CLASSIFICATION_PROMPT
        assert "Newsletter" in CLASSIFICATION_PROMPT


class TestExtractionPrompt:
    """Tests for the extraction prompt template."""

    def test_extraction_prompt_has_placeholders(self):
        """Extraction prompt should contain required placeholders."""
        assert "{subject}" in EXTRACTION_PROMPT
        assert "{body}" in EXTRACTION_PROMPT
        assert "{from_address}" in EXTRACTION_PROMPT

    def test_extraction_prompt_requests_fields(self):
        """Extraction prompt should request required fields."""
        assert "company" in EXTRACTION_PROMPT
        assert "position" in EXTRACTION_PROMPT
        assert "recruiter_name" in EXTRACTION_PROMPT
        assert "location" in EXTRACTION_PROMPT
