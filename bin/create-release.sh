#!/bin/bash
# Create a new release with auto-generated notes
# Usage: ./create-release.sh v3.0.0

set -e

VERSION=${1:-}

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v3.0.0"
    exit 1
fi

echo "🐣 Preparing release $VERSION..."

# Build binaries
echo "Building release binaries..."
bash bin/build-release.sh "$VERSION"

# Create git tag
echo "Creating git tag..."
git tag -a "$VERSION" -m "Release $VERSION"
git push origin "$VERSION"

echo ""
echo "✅ Tag created and pushed!"
echo ""
echo "📝 Next steps:"
echo "1. Go to: https://github.com/Josepavese/nido/releases/new?tag=$VERSION"
echo "2. Click 'Generate release notes' (GitHub will auto-generate from commits)"
echo "3. Add the Nido personality:"
echo "   - Add emoji to section headers"
echo "   - Add a witty intro paragraph"
echo "   - Highlight breaking changes with bird puns"
echo "4. Upload binaries from dist/ folder"
echo "5. Publish!"
echo ""
echo "🪺 Happy releasing!"
