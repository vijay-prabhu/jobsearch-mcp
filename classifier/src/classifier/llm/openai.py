"""OpenAI LLM provider implementation."""

import json
import logging
import os
from typing import Optional

from openai import AsyncOpenAI

from ..prompts import CLASSIFICATION_PROMPT, VALIDATION_PROMPT
from .base import ClassificationResult, LLMProvider, ValidationResult

logger = logging.getLogger(__name__)


class OpenAIProvider(LLMProvider):
    """OpenAI-based LLM provider."""

    def __init__(
        self,
        model: str = "gpt-4o-mini",
        api_key: Optional[str] = None,
    ):
        self.model = model
        self._api_key = api_key or os.environ.get("OPENAI_API_KEY")

        if not self._api_key:
            logger.warning("OPENAI_API_KEY not set - OpenAI provider will not work")
            self._client = None
        else:
            self._client = AsyncOpenAI(api_key=self._api_key)

    async def health_check(self) -> bool:
        """Check if OpenAI API is accessible."""
        if not self._client:
            return False

        try:
            # Simple API check - list models
            models = await self._client.models.list()
            return len(models.data) > 0
        except Exception as e:
            logger.error(f"OpenAI health check failed: {e}")
            return False

    async def classify(
        self,
        subject: str,
        body: str,
        from_address: str,
    ) -> ClassificationResult:
        """Classify an email using OpenAI."""
        if not self._client:
            return self._fallback_result("OpenAI API key not configured")

        # Truncate body if too long (OpenAI has larger context)
        max_body_length = 4000
        if len(body) > max_body_length:
            body = body[:max_body_length] + "..."

        prompt = CLASSIFICATION_PROMPT.format(
            subject=subject,
            body=body,
            from_address=from_address,
        )

        try:
            response = await self._client.chat.completions.create(
                model=self.model,
                messages=[{"role": "user", "content": prompt}],
                response_format={"type": "json_object"},
                temperature=0.1,
            )

            content = response.choices[0].message.content
            return self._parse_response(content)

        except Exception as e:
            logger.error(f"OpenAI classification failed: {e}")
            return self._fallback_result(str(e))

    def _parse_response(self, content: str) -> ClassificationResult:
        """Parse the LLM response into a ClassificationResult."""
        try:
            data = json.loads(content)

            return ClassificationResult(
                is_job_related=data.get("is_job_related", False),
                confidence=float(data.get("confidence", 0.0)),
                company=data.get("company"),
                position=data.get("position"),
                recruiter_name=data.get("recruiter_name"),
                classification=data.get("classification"),
                reasoning=data.get("reasoning"),
            )
        except (json.JSONDecodeError, KeyError, TypeError) as e:
            logger.warning(f"Failed to parse OpenAI response: {e}")
            return self._fallback_result(f"Parse error: {e}")

    def _fallback_result(self, reason: str) -> ClassificationResult:
        """Return a conservative fallback result."""
        return ClassificationResult(
            is_job_related=False,
            confidence=0.0,
            reasoning=f"Classification failed: {reason}",
        )

    async def validate(
        self,
        subject: str,
        body: str,
        from_address: str,
    ) -> ValidationResult:
        """Validate an email classification using structured multi-signal questions."""
        if not self._client:
            return self._fallback_validation_result("OpenAI API key not configured")

        max_body_length = 3000
        if len(body) > max_body_length:
            body = body[:max_body_length] + "..."

        prompt = VALIDATION_PROMPT.format(
            subject=subject,
            body=body,
            from_address=from_address,
        )

        try:
            response = await self._client.chat.completions.create(
                model=self.model,
                messages=[{"role": "user", "content": prompt}],
                response_format={"type": "json_object"},
                temperature=0.1,
            )

            content = response.choices[0].message.content
            return self._parse_validation_response(content)

        except Exception as e:
            logger.error(f"OpenAI validation failed: {e}")
            return self._fallback_validation_result(str(e))

    def _parse_validation_response(self, content: str) -> ValidationResult:
        """Parse the LLM response into a ValidationResult."""
        try:
            data = json.loads(content)

            return ValidationResult(
                is_direct_opportunity=data.get("is_direct_opportunity", False),
                is_recruiter_outreach=data.get("is_recruiter_outreach", False),
                is_interview_related=data.get("is_interview_related", False),
                is_job_alert_newsletter=data.get("is_job_alert_newsletter", False),
                is_marketing_promo=data.get("is_marketing_promo", False),
                is_application_response=data.get("is_application_response", False),
                final_verdict=data.get("final_verdict", False),
                confidence=float(data.get("confidence", 0.0)),
                reasoning=data.get("reasoning"),
            )
        except (json.JSONDecodeError, KeyError, TypeError) as e:
            logger.warning(f"Failed to parse validation response: {e}")
            return self._fallback_validation_result(f"Parse error: {e}")

    def _fallback_validation_result(self, reason: str) -> ValidationResult:
        """Return a conservative fallback validation result."""
        return ValidationResult(
            final_verdict=False,
            confidence=0.0,
            reasoning=f"Validation failed: {reason}",
        )

    async def close(self):
        """Close the client (no-op for OpenAI)."""
        pass
