"""FastAPI application for email classification."""

import logging
from contextlib import asynccontextmanager
from typing import Optional

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

from .llm import BatchClassificationResult, ClassificationResult, ValidationResult, get_provider

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Global providers (initialized on startup)
_providers = {}


class ClassifyRequest(BaseModel):
    """Request model for classification endpoint."""

    email_subject: str
    email_body: str
    email_from: str
    provider: str = "ollama"
    model: Optional[str] = None
    host: Optional[str] = None


class ValidateRequest(BaseModel):
    """Request model for validation endpoint."""

    email_subject: str
    email_body: str
    email_from: str
    provider: str = "ollama"
    model: Optional[str] = None
    host: Optional[str] = None


class EmailItem(BaseModel):
    """Single email in a batch request."""

    subject: str
    body: str
    from_address: str


class BatchClassifyRequest(BaseModel):
    """Request model for batch classification endpoint."""

    emails: list[EmailItem]
    provider: str = "ollama"
    model: Optional[str] = None
    host: Optional[str] = None


class HealthResponse(BaseModel):
    """Response model for health endpoint."""

    status: str
    ollama_available: bool
    openai_available: bool


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan - initialize and cleanup providers."""
    # Initialize default providers
    logger.info("Initializing LLM providers...")

    try:
        _providers["ollama"] = get_provider("ollama")
        ollama_ok = await _providers["ollama"].health_check()
        logger.info(f"Ollama provider: {'available' if ollama_ok else 'unavailable'}")
    except Exception as e:
        logger.warning(f"Failed to initialize Ollama: {e}")

    try:
        _providers["openai"] = get_provider("openai")
        openai_ok = await _providers["openai"].health_check()
        logger.info(f"OpenAI provider: {'available' if openai_ok else 'unavailable'}")
    except Exception as e:
        logger.warning(f"Failed to initialize OpenAI: {e}")

    yield

    # Cleanup
    for provider in _providers.values():
        if hasattr(provider, "close"):
            await provider.close()


app = FastAPI(
    title="JobSearch Classifier",
    description="Email classification service for job search tracking",
    version="0.1.0",
    lifespan=lifespan,
)


@app.get("/health", response_model=HealthResponse)
async def health():
    """Health check endpoint."""
    ollama_ok = False
    openai_ok = False

    if "ollama" in _providers:
        try:
            ollama_ok = await _providers["ollama"].health_check()
        except Exception:
            pass

    if "openai" in _providers:
        try:
            openai_ok = await _providers["openai"].health_check()
        except Exception:
            pass

    return HealthResponse(
        status="ok",
        ollama_available=ollama_ok,
        openai_available=openai_ok,
    )


@app.post("/classify", response_model=ClassificationResult)
async def classify(request: ClassifyRequest):
    """Classify an email and extract job-related information."""
    provider_name = request.provider

    # Get or create provider
    if provider_name in _providers:
        provider = _providers[provider_name]
    else:
        try:
            kwargs = {}
            if request.model:
                kwargs["model"] = request.model
            if request.host:
                kwargs["host"] = request.host
            provider = get_provider(provider_name, **kwargs)
        except ValueError as e:
            raise HTTPException(status_code=400, detail=str(e))

    # Check provider health
    try:
        is_healthy = await provider.health_check()
        if not is_healthy:
            raise HTTPException(
                status_code=503,
                detail=f"Provider '{provider_name}' is not available",
            )
    except Exception as e:
        raise HTTPException(
            status_code=503,
            detail=f"Provider health check failed: {e}",
        )

    # Classify the email
    try:
        result = await provider.classify(
            subject=request.email_subject,
            body=request.email_body,
            from_address=request.email_from,
        )
        return result
    except Exception as e:
        logger.error(f"Classification failed: {e}")
        raise HTTPException(
            status_code=500,
            detail=f"Classification failed: {e}",
        )


@app.post("/validate", response_model=ValidationResult)
async def validate(request: ValidateRequest):
    """Validate an email classification with structured multi-signal questions."""
    provider_name = request.provider

    # Get or create provider
    if provider_name in _providers:
        provider = _providers[provider_name]
    else:
        try:
            kwargs = {}
            if request.model:
                kwargs["model"] = request.model
            if request.host:
                kwargs["host"] = request.host
            provider = get_provider(provider_name, **kwargs)
        except ValueError as e:
            raise HTTPException(status_code=400, detail=str(e))

    # Check provider health
    try:
        is_healthy = await provider.health_check()
        if not is_healthy:
            raise HTTPException(
                status_code=503,
                detail=f"Provider '{provider_name}' is not available",
            )
    except Exception as e:
        raise HTTPException(
            status_code=503,
            detail=f"Provider health check failed: {e}",
        )

    # Validate the email
    try:
        result = await provider.validate(
            subject=request.email_subject,
            body=request.email_body,
            from_address=request.email_from,
        )
        return result
    except Exception as e:
        logger.error(f"Validation failed: {e}")
        raise HTTPException(
            status_code=500,
            detail=f"Validation failed: {e}",
        )


@app.post("/classify-batch", response_model=BatchClassificationResult)
async def classify_batch(request: BatchClassifyRequest):
    """Classify multiple emails in a single LLM call for better performance."""
    provider_name = request.provider

    # Get or create provider
    if provider_name in _providers:
        provider = _providers[provider_name]
    else:
        try:
            kwargs = {}
            if request.model:
                kwargs["model"] = request.model
            if request.host:
                kwargs["host"] = request.host
            provider = get_provider(provider_name, **kwargs)
        except ValueError as e:
            raise HTTPException(status_code=400, detail=str(e))

    # Check provider health
    try:
        is_healthy = await provider.health_check()
        if not is_healthy:
            raise HTTPException(
                status_code=503,
                detail=f"Provider '{provider_name}' is not available",
            )
    except Exception as e:
        raise HTTPException(
            status_code=503,
            detail=f"Provider health check failed: {e}",
        )

    # Convert request emails to dict format
    emails = [
        {"subject": e.subject, "body": e.body, "from_address": e.from_address}
        for e in request.emails
    ]

    # Classify the batch
    try:
        result = await provider.classify_batch(emails)
        return result
    except Exception as e:
        logger.error(f"Batch classification failed: {e}")
        raise HTTPException(
            status_code=500,
            detail=f"Batch classification failed: {e}",
        )


@app.get("/")
async def root():
    """Root endpoint with API information."""
    return {
        "name": "JobSearch Classifier",
        "version": "0.3.0",
        "endpoints": {
            "/health": "Health check",
            "/classify": "Email classification (POST)",
            "/classify-batch": "Batch email classification (POST)",
            "/validate": "Email validation with multi-signal analysis (POST)",
        },
    }
