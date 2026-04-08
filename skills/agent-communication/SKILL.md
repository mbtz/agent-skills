---
name: agent-communication
description: "Use when sending messages between agents, coordinating multi-agent tasks, or checking for replies via fmail."
metadata:
  short-description: Agent messaging via fmail
---

# Agent Communication

Use `fmail` for agent-to-agent messaging (not MCP agent mail). This skill covers registration, sending, and monitoring messages.

## Setup

1. Set your agent name: `export FMAIL_AGENT=<your-name>` (use a stable kebab-case name).
2. Register: `fmail register` to claim a unique name.

## Sending Messages

- **Direct message**: `fmail send @agent-name "your message"`
- **Topic message**: `fmail send topic-name "your message"` (e.g., `status`, `editing`)

## Reading and Monitoring

- **Read DMs**: `fmail log @agent-name -n 20`
- **Wait for reply**: `fmail watch topic-name --count 1`

## Example: Coordinating a Handoff

```bash
# Register yourself
export FMAIL_AGENT=code-reviewer
fmail register

# Ask another agent to review
fmail send @test-runner "Tests passed on feature branch, ready for deploy check"

# Wait for the response
fmail watch @test-runner --count 1

# Check recent conversation
fmail log @test-runner -n 5
```

## Key Rules

- Always register before sending messages.
- Use direct messages for 1:1 coordination and topics for broadcast status updates.
- Check logs regularly when waiting on another agent's response.

Reference: [fmail-quickref.md](references/fmail-quickref.md)
