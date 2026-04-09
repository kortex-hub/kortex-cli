# Website Documentation

This directory contains the MkDocs configuration for automatically generating the kdn documentation website from the repository's README.md.

## How It Works

1. **Single Source**: The project `README.md` (in the repository root) is the only file you edit
2. **Automatic Splitting**: The `hooks.py` script splits README.md by `##` headings into separate pages
3. **Build & Deploy**: GitHub Actions automatically builds and deploys the site on merge to main

## Local Preview

```bash
# From the website/ directory:
cd website

# Create and activate virtual environment (first time only)
python3 -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install dependencies (first time only)
pip install -r requirements.txt

# Start preview server
mkdocs serve

# Open http://127.0.0.1:8000 in browser
```

## Files

- **mkdocs.yml** - MkDocs configuration (theme, plugins, SEO settings)
- **hooks.py** - Custom hook that splits README.md into multiple pages
- **requirements.txt** - Python dependencies (MkDocs, Material theme, plugins)
- **docs/** - Temporary directory for generated markdown files (ignored by git)

## Deployment

The website is automatically deployed to GitHub Pages when changes are pushed to the `main` branch.

Website URL: https://openkaiden.github.io/kdn/
