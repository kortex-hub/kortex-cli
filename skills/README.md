# Agent Skills

This directory contains reusable skills that can be discovered and executed by any AI agent.

## Structure

Each skill is contained in its own subdirectory with a `SKILL.md` file that defines:
- The skill name and description
- Input parameters and argument hints
- Detailed instructions for execution
- Usage examples

## Available Skills

- **commit**: Generate conventional commit messages based on staged changes
- **copyright-headers**: Add or update Apache License 2.0 copyright headers to source files

## Usage

Agents can discover skills by scanning this directory for `SKILL.md` files. Each skill's metadata and instructions are contained in its respective file.
