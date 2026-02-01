"""Prompt templates for email classification."""

CLASSIFICATION_PROMPT = """Analyze this email and determine if it's job-related.

Email Details:
- Subject: {subject}
- From: {from_address}
- Body: {body}

Respond with a JSON object containing:
{{
    "is_job_related": true/false,
    "confidence": 0.0-1.0,
    "company": "company name if identified, null otherwise",
    "position": "job title if mentioned, null otherwise",
    "recruiter_name": "recruiter's name if identifiable, null otherwise",
    "classification": "see classification guidelines below",
    "reasoning": "brief explanation of your decision"
}}

Classification guidelines:
- recruiter_outreach: Initial contact from a recruiter about an opportunity
- application_confirmation: Automated confirmation that application was received
- interview_request: Request to schedule an interview or call
- rejection: Notification that you were not selected
- offer: Job offer or compensation discussion
- follow_up: Continued conversation in an existing thread
- other: Job-related but doesn't fit above categories

Be conservative with confidence scores. If uncertain, set is_job_related to false.

EXCLUDE these types (set is_job_related to false):
- Job alert digests or "new jobs for you" emails
- LinkedIn notifications (connection requests, who viewed your profile, etc.)
- Newsletter or marketing emails
- Recruiter spam (mass outreach with no specific role)
- Promotional emails from job boards

INCLUDE these types (set is_job_related to true):
- Direct recruiter outreach mentioning a specific role or company
- Application confirmations from companies you applied to
- Interview scheduling or rescheduling
- Rejection letters
- Offer letters or compensation discussions
- Follow-up emails in an ongoing conversation

JSON response:"""

EXTRACTION_PROMPT = """Extract structured information from this job-related email.

Email Details:
- Subject: {subject}
- From: {from_address}
- Body: {body}

Extract and return a JSON object with:
{{
    "company": "The company name (not the ATS platform)",
    "position": "The job title/position",
    "recruiter_name": "The recruiter's name (first and last if available)",
    "recruiter_title": "The recruiter's job title if mentioned",
    "location": "Job location if mentioned",
    "salary_range": "Salary range if mentioned",
    "next_steps": "Any mentioned next steps or action items",
    "deadline": "Any mentioned deadline or timeline"
}}

For each field, use null if the information is not clearly stated in the email.
Be precise - only extract information that is explicitly mentioned.

JSON response:"""

BATCH_CLASSIFICATION_PROMPT = """Analyze these {count} emails and determine if each is job-related.

{emails}

For EACH email, respond with a JSON array containing one object per email in order:
[
  {{
    "index": 0,
    "is_job_related": true/false,
    "confidence": 0.0-1.0,
    "company": "company name or null",
    "position": "job title or null",
    "recruiter_name": "name or null",
    "classification": "recruiter_outreach|application_confirmation|interview_request|rejection|offer|follow_up|other",
    "reasoning": "brief explanation"
  }},
  ...
]

Classification guidelines:
- recruiter_outreach: Initial contact from a recruiter about an opportunity
- application_confirmation: Automated confirmation that application was received
- interview_request: Request to schedule an interview or call
- rejection: Notification that you were not selected
- offer: Job offer or compensation discussion
- follow_up: Continued conversation in an existing thread
- other: Job-related but doesn't fit above categories

EXCLUDE (is_job_related=false):
- Job alert digests or "new jobs for you" emails
- LinkedIn notifications (connection requests, who viewed profile)
- Newsletter or marketing emails
- Recruiter spam (mass outreach with no specific role)
- Promotional emails from job boards

INCLUDE (is_job_related=true):
- Direct recruiter outreach with specific role/company
- Application confirmations
- Interview scheduling
- Rejection/offer letters

Be conservative - when uncertain, set is_job_related to false.

JSON array response:"""

VALIDATION_PROMPT = """You are validating whether an email is genuinely job-search related.

Email Details:
- Subject: {subject}
- From: {from_address}
- Body preview: {body}

Answer these questions about the email. Respond with a JSON object:
{{
    "is_direct_opportunity": true/false,
    "is_recruiter_outreach": true/false,
    "is_interview_related": true/false,
    "is_job_alert_newsletter": true/false,
    "is_marketing_promo": true/false,
    "is_application_response": true/false,
    "final_verdict": true/false,
    "confidence": 0.0-1.0,
    "reasoning": "brief explanation"
}}

Definitions:
- is_direct_opportunity: Email about a specific job opening at a specific company
- is_recruiter_outreach: Personal message from a recruiter (not automated)
- is_interview_related: Scheduling, confirming, or following up on an interview
- is_job_alert_newsletter: Automated digest of job listings (e.g., "10 new jobs for you")
- is_marketing_promo: Promotional content, newsletters, or marketing emails
- is_application_response: Confirmation, rejection, or update about YOUR application

Rules for final_verdict:
- TRUE if: is_direct_opportunity OR is_recruiter_outreach OR is_interview_related OR is_application_response
- FALSE if: is_job_alert_newsletter OR is_marketing_promo
- When signals conflict, lean towards FALSE (conservative)

IMPORTANT: Mass recruiter spam (generic "exciting opportunity" with no specific role) should be:
- is_recruiter_outreach: false (not personal)
- is_marketing_promo: true
- final_verdict: false

JSON response:"""
