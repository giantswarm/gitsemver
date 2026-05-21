LDFLAGS = -w -linkmode 'auto' -extldflags '$(EXTLDFLAGS)' \
  -X '$(MODULE)/pkg/project.buildTimestamp=${BUILDTIMESTAMP}' \
  -X '$(MODULE)/pkg/project.gitSHA=${GITSHA1}' \
  -X '$(MODULE)/pkg/project.version=${VERSION}'
