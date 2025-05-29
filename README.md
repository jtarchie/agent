# Agent

Agent is an AI-assisted development workflow tool that helps automate software
engineering tasks by breaking them down into planning and execution phases.

## Install

We distribute Linux and Mac [homebrew](https://brew.sh) support.

```bash
brew tap jtarchie/agent https://github.com/jtarchie/agent
brew install agent
```

## Overview

This project provides a command-line interface that leverages large language
models (LLMs) to assist with software development tasks. It works in two main
phases:

1. **Planning Phase** - A planning agent analyzes your request and files to
   create a detailed, step-by-step plan for addressing your task.

2. **Execution Phase** - An execution agent systematically works through the
   plan, using tools to read files, run terminal commands, and edit code.

## Features

- **Two-Phase Approach**: Separates planning from execution for methodical and
  traceable task completion
- **Multiple Tools**: Built-in capabilities for file operations, terminal
  commands, and code editing
- **Customizable Models**: Configure which AI models to use for planning and
  execution phases
- **Language Detection**: Automatically identifies programming languages for
  context-aware assistance

## How It Works

Agent analyzes your code files and request, then generates a comprehensive plan
that breaks down complex tasks into manageable steps. The execution agent then
follows this plan, using specialized tools to interact with your codebase and
development environment.

## Tools

The agent provides several tools for interacting with your development
environment:

- **ReadFile**: Reads specific lines from files in your codebase
- **RunInTerminal**: Executes terminal commands with explanations
- **InsertEditIntoFile**: Updates file content with proper tracking

## Architecture

The system is built around:

- A planning agent that uses the phi4-mini-reasoning model by default
- An execution agent that uses the qwen3:8b model by default
- Embedded templates for agent prompting
- A flexible tool system for codebase interaction

Agent is designed to follow software engineering best practices and serve as an
assistant rather than a replacement for human developers.

## License

MIT License - See LICENSE file for details.
