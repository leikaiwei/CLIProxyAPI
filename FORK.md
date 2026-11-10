# Fork releases

This fork carries changes awaiting upstream review. The `main` branch mirrors upstream, while the default `fork` branch contains the patches and this release workflow

`Fork release` checks the latest upstream stable release every two hours, at minute 17, and can also be started manually. It merges the release tag without resetting or force-pushing our changes. Merge conflicts stop the run for manual resolution. The workflow runs `go test ./...` and compiles the server before pushing the merged source. Scheduled runs skip the build when the public `stable` image already records the resulting source commit. Manual runs always build a new version, and failed publications are retried by the next scheduled run

Native amd64 and arm64 runners build the same source and verify startup, authenticated model listing, and unauthenticated rejection. Only after both images pass does the workflow publish `ghcr.io/leikaiwei/cliproxyapi:stable`. Version tags include the upstream release, workflow run ID, and attempt, such as `v8.0.63-fork.123456.1`. Each image records its source commit and upstream release in OCI labels

A failed merge, test, build, or smoke check leaves `stable` unchanged. A failed build can leave a tested source merge on `fork` and an unused architecture tag. Production should retain the previously deployed digest for rollback. The workflow does not access or deploy to any server

The workflow needs `contents: write` to save merges and `packages: write` to publish to GHCR. It uses the repository's `GITHUB_TOKEN`, with no registry password or server credentials. Set the package visibility to public before unauthenticated production pulls. Keep inherited upstream release workflows disabled in this fork, and enable only `Fork release`. Set `fork` as the repository default branch so its schedule runs

An upstream change to workflow files can require additional workflow-write permission when GitHub accepts the merge push. If GitHub rejects that push, synchronization stops before image publication. Review and apply that update with a maintainer credential rather than force-resetting the fork or bypassing the checks

The separate fork Dockerfile pins the Go toolchain and base images by digest. A future upstream Go requirement beyond that pinned toolchain intentionally fails the build until the toolchain image is reviewed and updated. Dependency checksums come from `go.sum`; model catalogs remain those tracked by the upstream release rather than being refreshed during the build

GitHub schedules can be delayed, and public repository schedules are disabled after 60 days without activity. Check workflow failures and the last successful image before changing a server's image source. Successful source synchronization does not imply a new `stable` image was published

Keep upstream pull requests limited to the functional patch. Once upstream includes the fix, verify the regression tests and remove any redundant local patch in a separate reviewed change
