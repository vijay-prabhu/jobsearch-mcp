"""Ollama LLM provider implementation."""

import json
import logging

import httpx

from ..prompts import CLASSIFICATION_PROMPT, VALIDATION_PROMPT
from .base import ClassificationResult, LLMProvider, ValidationResult

logger = logging.getLogger(__name__)


class OllamaProvider(LLMProvider):
    """Ollama-based LLM provider for local inference."""

    def __init__(
        self,
        model: str = "llama3.2:1b",
        host: str = "http://localhost:11434",
        timeout: float = 60.0,
    ):
        self.model = model
        self.host = host.rstrip("/")
        self.timeout = timeout
        self._client = httpx.AsyncClient(timeout=timeout)

    async def health_check(self) -> bool:
        """Check if Ollama is running and the model is available."""
        try:
            response = await self._client.get(f"{self.host}/api/tags")
            if response.status_code != 200:
                return False

            data = response.json()
            models = [m.get("name", "") for m in data.get("models", [])]

            # Check if our model is available (with or without tag)
            model_base = self.model.split(":")[0]
            for m in models:
                if m.startswith(model_base):
                    return True

            logger.warning(f"Model {self.model} not found. Available: {models}")
            return False
        except Exception as e:
            logger.error(f"Ollama health check failed: {e}")
            return False

    async def classify(
        self,
        subject: str,
        body: str,
        from_address: str,
    ) -> ClassificationResult:
        """Classify an email using Ollama."""
        # Truncate body if too long
        max_body_length = 2000
        if len(body) > max_body_length:
            body = body[:max_body_length] + "..."

        prompt = CLASSIFICATION_PROMPT.format(
            subject=subject,
            body=body,
            from_address=from_address,
        )

        try:
            response = await self._client.post(
                f"{self.host}/api/chat",
                json={
                    "model": self.model,
                    "messages": [{"role": "user", "content": prompt}],
                    "format": "json",
                    "stream": False,
                    "options": {
                        "temperature": 0.1,  # Low temperature for consistency
                    },
                },
            )
            response.raise_for_status()

            data = response.json()
            content = data.get("message", {}).get("content", "")

            return self._parse_response(content)

        except httpx.TimeoutException:
            logger.error("Ollama request timed out")
            return self._fallback_result("Request timed out")
        except Exception as e:
            logger.error(f"Ollama classification failed: {e}")
            return self._fallback_result(str(e))

    def _parse_response(self, content: str) -> ClassificationResult:
        """Parse the LLM response into a ClassificationResult."""
        try:
            # Try to extract JSON from the response
            content = content.strip()

            # Handle case where response might have extra text
            if not content.startswith("{"):
                start = content.find("{")
                if start != -1:
                    content = content[start:]

            if not content.endswith("}"):
                end = content.rfind("}")
                if end != -1:
                    content = content[: end + 1]

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
            logger.warning(f"Failed to parse Ollama response: {e}")
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
        # Truncate body if too long
        max_body_length = 1500
        if len(body) > max_body_length:
            body = body[:max_body_length] + "..."

        prompt = VALIDATION_PROMPT.format(
            subject=subject,
            body=body,
            from_address=from_address,
        )

        try:
            response = await self._client.post(
                f"{self.host}/api/chat",
                json={
                    "model": self.model,
                    "messages": [{"role": "user", "content": prompt}],
                    "format": "json",
                    "stream": False,
                    "options": {
                        "temperature": 0.1,
                    },
                },
            )
            response.raise_for_status()

            data = response.json()
            content = data.get("message", {}).get("content", "")

            return self._parse_validation_response(content)

        except httpx.TimeoutException:
            logger.error("Ollama validation request timed out")
            return self._fallback_validation_result("Request timed out")
        except Exception as e:
            logger.error(f"Ollama validation failed: {e}")
            return self._fallback_validation_result(str(e))

    def _parse_validation_response(self, content: str) -> ValidationResult:
        """Parse the LLM response into a ValidationResult."""
        try:
            content = content.strip()

            if not content.startswith("{"):
                start = content.find("{")
                if start != -1:
                    content = content[start:]

            if not content.endswith("}"):
                end = content.rfind("}")
                if end != -1:
                    content = content[: end + 1]

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
        """Close the HTTP client."""
        await self._client.aclose()
