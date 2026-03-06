#!/bin/bash
set -e

# Upload spored binaries to S3 with SHA256 checksums
# Usage: ./upload_spawnd.sh [aws-profile] [regions...]
#
# Examples:
#   ./upload_spawnd.sh my-spawn-account us-east-1 us-west-2
#   ./upload_spawnd.sh my-spawn-account all  # All regions

PROFILE=${1:-spore-host-infra}
shift

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SPAWN_BIN_DIR="$PROJECT_ROOT/spawn/bin"

# Check if binaries exist
if [ ! -f "$SPAWN_BIN_DIR/spored-linux-amd64" ]; then
    echo "❌ Error: spored binaries not found in $SPAWN_BIN_DIR"
    echo "Run 'make build-all' first to build binaries"
    exit 1
fi

# Default regions if not specified
if [ $# -eq 0 ]; then
    # Default: US regions only
    REGIONS=(
        us-east-1
        us-east-2
        us-west-1
        us-west-2
    )
elif [ "$1" = "all" ]; then
    # All major regions
    REGIONS=(
        us-east-1
        us-east-2
        us-west-1
        us-west-2
        eu-west-1
        eu-west-2
        eu-central-1
        ap-southeast-1
        ap-southeast-2
        ap-northeast-1
    )
else
    REGIONS=("$@")
fi

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Uploading spored binaries to S3"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Profile: $PROFILE"
echo "Regions: ${REGIONS[*]}"
echo ""

# Binaries to upload
BINARIES=(
    "spored-linux-amd64"
    "spored-linux-arm64"
)

# Generate SHA256 checksums
echo "→ Generating SHA256 checksums..."
cd "$SPAWN_BIN_DIR"

for binary in "${BINARIES[@]}"; do
    if [ ! -f "$binary" ]; then
        echo "  ⚠️  Warning: $binary not found, skipping"
        continue
    fi

    echo "  Computing: $binary"
    sha256sum "$binary" | awk '{print $1}' > "${binary}.sha256"

    # Display checksum
    checksum=$(cat "${binary}.sha256")
    size=$(ls -lh "$binary" | awk '{print $5}')
    echo "    SHA256: $checksum"
    echo "    Size:   $size"
done

echo ""

# Upload to each region
for region in "${REGIONS[@]}"; do
    bucket_name="spawn-binaries-${region}"

    echo "→ Uploading to ${bucket_name}"

    for binary in "${BINARIES[@]}"; do
        if [ ! -f "$binary" ]; then
            continue
        fi

        # Upload binary
        echo "  Uploading: $binary"
        aws s3 cp "$binary" \
            "s3://${bucket_name}/${binary}" \
            --profile "$PROFILE" \
            --region "$region" \
            --content-type "application/octet-stream" \
            --metadata "version=0.1.0,uploaded=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
            --no-progress

        # Upload checksum
        aws s3 cp "${binary}.sha256" \
            "s3://${bucket_name}/${binary}.sha256" \
            --profile "$PROFILE" \
            --region "$region" \
            --content-type "text/plain" \
            --no-progress

        # Get public URL
        url="https://${bucket_name}.s3.amazonaws.com/${binary}"
        echo "    URL: $url"
    done

    echo "  ✅ ${bucket_name} updated"
    echo ""
done

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  ✅ Upload complete!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Test download and verification:"
echo "  curl -O https://spawn-binaries-us-east-1.s3.amazonaws.com/spored-linux-amd64"
echo "  curl -O https://spawn-binaries-us-east-1.s3.amazonaws.com/spored-linux-amd64.sha256"
echo "  echo \"\$(cat spored-linux-amd64.sha256)  spored-linux-amd64\" | sha256sum --check"
echo ""

# Display checksums for reference
echo "SHA256 Checksums:"
for binary in "${BINARIES[@]}"; do
    if [ -f "${binary}.sha256" ]; then
        checksum=$(cat "${binary}.sha256")
        echo "  $binary: $checksum"
    fi
done
echo ""
