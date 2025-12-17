# Contributing to actionsum

Thank you for your interest in contributing to actionsum! This document provides guidelines for contributing to the project.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/actionsum.git`
3. Create a feature branch: `git checkout -b feature/your-feature-name`
4. Make your changes
5. Run tests: `make test`
6. Commit your changes: `git commit -m "feat: add amazing feature"`
7. Push to your fork: `git push origin feature/your-feature-name`
8. Open a Pull Request

## Development Setup

### Prerequisites
- Go 1.21 or later
- Linux with X11 or Wayland
- `xdotool` (for X11 development)
- `gdbus` (for Wayland/GNOME development)

### Building
```bash
make build
```

### Running Tests
```bash
make test              # Run all tests
make test-verbose      # Run with verbose output
make test-coverage     # Generate coverage report
```

### Running Locally
```bash
go run cmd/actionsum/main.go serve
```

## Code Style

- Follow standard Go conventions
- Run `gofmt` before committing
- Use meaningful variable and function names
- Add comments for exported functions and types
- Keep functions focused and small

## Commit Messages

We follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` - New features
- `fix:` - Bug fixes
- `docs:` - Documentation changes
- `test:` - Test additions or modifications
- `refactor:` - Code refactoring
- `chore:` - Maintenance tasks

Example:
```
feat: add support for Hyprland compositor
fix: resolve race condition in tracker service
docs: update installation instructions
```

## Pull Request Process

1. Update the README.md with details of changes if applicable
2. Update tests to cover your changes
3. Ensure all tests pass
4. Update documentation if you're changing functionality
5. The PR will be merged once approved by a maintainer

## Testing

- Write unit tests for new features
- Ensure existing tests pass
- Test on both X11 and Wayland if possible
- Test on different distributions if possible

## Questions?

Feel free to open an issue for:
- Bug reports
- Feature requests
- Questions about the codebase
- Suggestions for improvements

Thank you for contributing! 🎉
