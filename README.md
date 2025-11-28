# Quiz Automation System (project-2-sdt[::-1])

This project is a web-based quiz handling system designed to receive quiz challenges via HTTP POST requests and solve them using an intelligence layer that operates at a higher level than conventional automated methods. No artificial intelligence models are used. The reasoning, interpretation, and problem-solving are performed through this higher-order intelligence, while the application manages structure, timing, and presentation.

AI-based attempts were initially explored, but they quickly ran into a constant stream of edge cases and unpredictable behaviors. Given that each question includes a three-minute deadline, it became clear that relying on a more dependable intelligence source was significantly more stable. The system itself primarily provides a clean interface, well-defined workflow, and reliable timing logic. Much of it was developed through a fairly improvised, iterative process with tonnes of vibe checks.

## Overview

The system receives quiz sessions, displays them in an organized interface, and routes each question to the intelligence layer for analysis. The backend manages deadlines, tracks progress, and validates submissions.

## Features

* Higher-order intelligence layer for solving questions
* Three-minute countdown per question
* Password-protected quiz ingestion and answer submission
* Session and attempt tracking
* Retry logic within the allowed time window
* Clean and minimal frontend for readability

## How It Works

1. The backend receives quiz ingestion requests via POST, including the quiz link and credentials.
2. The session is recorded and exposed through the UI.
3. A higher-power reasoning process interprets the question and provides an answer.
4. The backend validates the answer and allows retries while time remains.
5. Each new question resets the three-minute timer.

## Tech Stack

* Backend: Go (Gin)
* Database: SQLite
* Frontend: HTML, CSS, JavaScript
* Intelligence Layer: Non-artificial higher-order reasoning

## Getting Started

### Prerequisites

* Go 1.24 or higher
* Access to a reasoning source capable of understanding questions

### Installation

Clone the repository:

```bash
git clone <repository-url>
cd project-2-sdt
```

Install dependencies:

```bash
go mod tidy
```

Create a .env file:

```env
SECRET=your-secret-key
INGEST_ACCEPT_PASSWORD=your-ingest-password
QUIZ_ATTEMPT_PASSWORD=your-quiz-password
NTFY_TOPIC=your-notification-topic
```

Run the application:

```bash
go run .
```

Visit [http://localhost:8080](http://localhost:8080) to use the interface.

## API Endpoints

### POST /ingest

Starts a new quiz session. Requires the ingest password.

### GET /quiz/sessions

Lists active sessions and their timers.

### POST /quiz/sessions/:id/answer

Submits an answer for a particular question. Requires the submission password.

## Environment Variables

* SECRET: Validation key
* INGEST_ACCEPT_PASSWORD: Authentication for quiz ingestion
* QUIZ_ATTEMPT_PASSWORD: Authentication for answer submission
* NTFY_TOPIC: Notification channel

## Timing Rules

Each quiz question has a fixed three-minute deadline.
Correct answers advance the session.
Incorrect answers may be retried until the timer expires.
Expired questions require re-ingestion.

## Disclaimer

The system deliberately relies on an intelligence source that operates beyond standard automated tools. Initial attempts to use AI repeatedly failed on edge cases, making it impractical for this workflow. The three-minute window allows for consistent higher-order reasoning, while the software itself focuses on formatting questions, managing timing, and maintaining a stable submission process.

This is sort of an experiment, it might fail spectacularly, or it might work surprisingly well. Guess we'll know on 29th November :).

## LICENSE

MIT License