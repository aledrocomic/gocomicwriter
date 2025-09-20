# CI/CD on AWS for gocomic using GitHub as VCS

This guide walks you through setting up a production‑ready CI/CD pipeline for this Go CLI project (gocomic) with GitHub as your VCS and AWS for artifact hosting. It includes:

- Continuous Integration (build, vet, formatting, module hygiene)
- Release builds for multiple OS/architectures
- Publishing artifacts to GitHub Releases and optionally to an S3 bucket (via OIDC – no long‑lived AWS keys)
- An alternative AWS‑native setup using CodePipeline + CodeBuild

If you simply want a working pipeline fast, follow Steps 1–4 and use the provided GitHub Actions workflows.

---

## 0) What you’ll build

- CI on every pull request and push: verifies module integrity, formatting, vets code, and builds the binary
- CD on a git tag like `v0.1.0`: builds cross‑platform binaries, attaches them to a GitHub Release, and optionally mirrors them to S3

Project specifics:
- Module: `gocomic`
- Entry point: `./cmd/gocomic`
- Go version: `1.23`
- No tests at the moment; CI focuses on fast sanity checks

---

## 1) Prerequisites

- An AWS account and permissions to create IAM roles, S3 buckets, and optionally CloudFront distributions
- A GitHub repository containing this project
- Admin access to configure GitHub Actions and repository variables/secrets

---

## 2) Create an S3 bucket for release artifacts (optional but recommended)

If you want artifacts also available from AWS (in addition to GitHub Releases):

- Choose an S3 bucket name (e.g., `gocomic-artifacts-<account-id>-<region>`)
- Create the bucket in your preferred region
- If you plan to serve files publicly, enable public read for objects via a bucket policy or serve via CloudFront. Example minimal bucket policy (public read of objects):

```
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "PublicReadGetObject",
      "Effect": "Allow",
      "Principal": "*",
      "Action": "s3:GetObject",
      "Resource": "arn:aws:s3:::gocomic-artifacts-<account-id>-<region>/*"
    }
  ]
}
```

Tip: Prefer CloudFront for production distribution and keep the bucket private, granting the CI role write access and CloudFront the read origin access.

---

## 3) Configure GitHub OIDC access to AWS (no long‑lived keys)

Using GitHub’s OIDC provider allows GitHub Actions to assume an IAM role in your AWS account without storing static AWS keys.

### 3.1 Add GitHub OIDC provider (one‑time per account)

In IAM → Identity providers → Add provider:
- Provider URL: `https://token.actions.githubusercontent.com`
- Audience: `sts.amazonaws.com`

If your account already has this provider, you can reuse it.

### 3.2 Create an IAM role for GitHub Actions

- IAM → Roles → Create role → Web identity
- Identity provider: `token.actions.githubusercontent.com`
- Audience: `sts.amazonaws.com`
- Add a trust policy that limits which repo/branches/tags can assume the role (replace `OWNER` and `REPO`):

```
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::<ACCOUNT_ID>:oidc-provider/token.actions.githubusercontent.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "token.actions.githubusercontent.com:aud": "sts.amazonaws.com"
        },
        "StringLike": {
          "token.actions.githubusercontent.com:sub": [
            "repo:OWNER/REPO:ref:refs/heads/main",
            "repo:OWNER/REPO:ref:refs/heads/*",
            "repo:OWNER/REPO:ref:refs/tags/v*"
          ]
        }
      }
    }
  ]
}
```

- Attach a least‑privilege permissions policy. For S3 publishing and optional CloudFront invalidation:

```
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "WriteArtifactsToBucket",
      "Effect": "Allow",
      "Action": ["s3:PutObject", "s3:PutObjectAcl", "s3:DeleteObject"],
      "Resource": "arn:aws:s3:::gocomic-artifacts-<account-id>-<region>/*"
    },
    {
      "Sid": "ListBucket",
      "Effect": "Allow",
      "Action": ["s3:ListBucket"],
      "Resource": "arn:aws:s3:::gocomic-artifacts-<account-id>-<region>"
    },
    {
      "Sid": "InvalidateCloudFront",
      "Effect": "Allow",
      "Action": ["cloudfront:CreateInvalidation"],
      "Resource": "*"
    }
  ]
}
```

Record the role ARN, e.g., `arn:aws:iam::<ACCOUNT_ID>:role/github-actions-gocomic`.

---

## 4) Configure GitHub repository variables

In GitHub → Settings → Secrets and variables → Actions:

- Secrets
  - `AWS_ROLE_ARN` = your IAM role ARN
- Variables (or use Secrets if you prefer)
  - `AWS_REGION` = e.g., `eu-central-1`
  - `S3_BUCKET` = your S3 bucket name (if using S3 publish)

Optionally protect the `main` branch and require the CI workflow to pass before merging.

---

## 5) Add the CI workflow (build, vet, formatting, tidy check)

A workflow is provided at `.github/workflows/ci.yml`. It runs on pushes and PRs and will:
- Set up Go 1.23
- Cache modules
- Ensure `go mod tidy` would not change files
- Fail on formatting diffs
- `go vet`
- Build the project

If you need additional linters (e.g., staticcheck), you can add them later.

---

## 6) Add the Release workflow (build multi‑platform and publish)

A workflow is provided at `.github/workflows/release.yml`. It triggers on tags starting with `v`, e.g., `v0.1.0` and will:
- Cross‑compile for Linux, Windows, and macOS (amd64 and arm64 where applicable)
- Package artifacts
- Create or update a GitHub Release and attach artifacts
- Optionally publish artifacts to S3 under `s3://$S3_BUCKET/releases/<tag>/`

Note on versioning: the project currently has a compile‑time constant `internal/version.Version`. To change the printed version, bump it in code before tagging a release. The workflow does not attempt to override it via ldflags because it’s declared as a const.

---

## 7) Cut your first release

1) Update `internal/version/version.go` with the new version string
2) Commit and push to `main`
3) Create a tag and push it:

```
# choose your version
VER=v0.1.0

git tag "$VER"
git push origin "$VER"
```

4) Watch the Actions tab: the release workflow should build, publish a GitHub Release, and (optionally) upload to S3

---

## 8) (Optional) AWS‑native alternative: CodePipeline + CodeBuild

If you prefer AWS to orchestrate builds/deployments, you can:

- Create a CodePipeline with GitHub (v2) as the source (connect your GitHub repo)
- Add a CodeBuild project with the following `buildspec.yml` at the repository root or point CodeBuild to the inline spec

Example `buildspec.yml` (build and publish to S3 on tags):

```
version: 0.2
env:
  variables:
    ARTIFACT_DIR: dist
phases:
  install:
    runtime-versions:
      golang: 1.23
  pre_build:
    commands:
      - go version
      - go mod download
      - go vet ./...
      - echo "Checking formatting..."
      - |
        DIFF=$(gofmt -s -l .)
        if [ -n "$DIFF" ]; then
          echo "Files need gofmt:" && echo "$DIFF" && exit 1
        fi
  build:
    commands:
      - mkdir -p "$ARTIFACT_DIR"
      - echo "Building for linux/amd64..."
      - GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o "$ARTIFACT_DIR/gocomic-linux-amd64" ./cmd/gocomic
      - echo "Building for windows/amd64..."
      - GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o "$ARTIFACT_DIR/gocomic-windows-amd64.exe" ./cmd/gocomic
  post_build:
    commands:
      - |
        if [[ "$CODEBUILD_RESOLVED_SOURCE_VERSION" =~ ^refs/tags/ ]]; then
          TAG=${CODEBUILD_RESOLVED_SOURCE_VERSION#refs/tags/}
        else
          TAG=build-${CODEBUILD_BUILD_NUMBER}
        fi
        echo "Tag: $TAG"
      - |
        if [ -n "$S3_BUCKET" ]; then
          aws s3 sync "$ARTIFACT_DIR/" "s3://$S3_BUCKET/releases/$TAG/" --acl public-read
        else
          echo "S3_BUCKET not set; skipping upload"
        fi
artifacts:
  files:
    - dist/**/*
```

In CodeBuild environment variables, set `S3_BUCKET` and assign an IAM role with S3 write permissions similar to the policy above.

---

## 9) Operations, rollbacks, and security

- Branch protections: require the CI workflow on `main`
- Environments: use GitHub Environments (e.g., `prod`) to gate the release job with approvals
- Rollback: create a new tag pointing to the previous commit; the workflow will republish artifacts for that tag
- Least privilege: scope the IAM role trust policy to your repo/branches/tags; scope permissions only to your S3 bucket and optional CloudFront distribution
- SBOM/signing: consider adding `go version` metadata, SBOM (e.g., `syft`), and signing (e.g., Sigstore/cosign) later

---

## 10) Files added by this guide

- `.github/workflows/ci.yml` — CI on push/PR
- `.github/workflows/release.yml` — Release on tags, publish to GitHub Releases and optional S3

You can tailor the matrices, add more linters, or integrate other distribution channels as needed.
