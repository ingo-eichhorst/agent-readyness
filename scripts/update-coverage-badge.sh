#!/bin/bash
# Script to calculate test coverage and show how to update the README badge

set -e

echo "📊 Calculating test coverage..."
echo ""

# Run tests with coverage
go test ./... -coverprofile=cover.out > /dev/null 2>&1

# Get total coverage percentage
COVERAGE=$(go tool cover -func=cover.out | grep total | awk '{print $3}' | sed 's/%//')

echo "✅ Total Coverage: ${COVERAGE}%"
echo ""
echo "📝 To update the README badge:"
echo ""
echo "   Current badge:"
echo "   [![Coverage](https://img.shields.io/badge/coverage-XX.X%25-brightgreen)](https://github.com/ingo-eichhorst/agent-readyness)"
echo ""
echo "   New badge:"
echo "   [![Coverage](https://img.shields.io/badge/coverage-${COVERAGE}%25-brightgreen)](https://github.com/ingo-eichhorst/agent-readyness)"
echo ""

# Determine color based on coverage
if (( $(echo "$COVERAGE >= 80" | bc -l) )); then
    COLOR="brightgreen"
elif (( $(echo "$COVERAGE >= 60" | bc -l) )); then
    COLOR="green"
elif (( $(echo "$COVERAGE >= 40" | bc -l) )); then
    COLOR="yellow"
else
    COLOR="red"
fi

echo "   Recommended color: $COLOR"
echo ""
echo "📋 Coverage by package:"
echo ""
go tool cover -func=cover.out | grep -v "total:" | awk '{printf "   %-50s %s\n", $1":"$2, $3}'
echo ""
echo "🌐 Generate HTML report:"
echo "   go tool cover -html=cover.out -o coverage.html"
echo "   open coverage.html"
