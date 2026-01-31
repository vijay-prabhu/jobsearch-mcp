"""Tests for the FastAPI application endpoints."""

import pytest
from httpx import ASGITransport, AsyncClient

from classifier.main import app


@pytest.fixture
async def client():
    """Create an async test client."""
    async with AsyncClient(
        transport=ASGITransport(app=app),
        base_url="http://test",
    ) as ac:
        yield ac


class TestRootEndpoint:
    """Tests for the root endpoint."""

    async def test_root_returns_api_info(self, client):
        """Root endpoint should return API information."""
        response = await client.get("/")
        assert response.status_code == 200

        data = response.json()
        assert data["name"] == "JobSearch Classifier"
        assert data["version"] == "0.1.0"
        assert "/health" in data["endpoints"]
        assert "/classify" in data["endpoints"]


class TestHealthEndpoint:
    """Tests for the health check endpoint."""

    async def test_health_returns_status(self, client):
        """Health endpoint should return status information."""
        response = await client.get("/health")
        assert response.status_code == 200

        data = response.json()
        assert data["status"] == "ok"
        assert "ollama_available" in data
        assert "openai_available" in data


class TestClassifyEndpoint:
    """Tests for the classification endpoint."""

    async def test_classify_rejects_invalid_provider(self, client):
        """Classification should reject unknown provider."""
        response = await client.post(
            "/classify",
            json={
                "email_subject": "Test",
                "email_body": "Test body",
                "email_from": "test@example.com",
                "provider": "nonexistent",
            },
        )
        assert response.status_code == 400

    async def test_classify_requires_subject(self, client):
        """Classification requires email_subject field."""
        response = await client.post(
            "/classify",
            json={
                "email_body": "Test body",
                "email_from": "test@example.com",
            },
        )
        assert response.status_code == 422  # Validation error

    async def test_classify_requires_body(self, client):
        """Classification requires email_body field."""
        response = await client.post(
            "/classify",
            json={
                "email_subject": "Test",
                "email_from": "test@example.com",
            },
        )
        assert response.status_code == 422

    async def test_classify_requires_from(self, client):
        """Classification requires email_from field."""
        response = await client.post(
            "/classify",
            json={
                "email_subject": "Test",
                "email_body": "Test body",
            },
        )
        assert response.status_code == 422
