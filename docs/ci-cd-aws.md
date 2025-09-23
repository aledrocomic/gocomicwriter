# AWS CI/CD with CodePipeline for gocomicwriter

This document describes how to set up CI/CD for this repository on AWS using the provided CloudFormation template at docs/aws-codepipeline.yml. It assumes you want an AWS‑native pipeline (CodePipeline + CodeBuild) with GitHub (via CodeStar Connections) as the source.

What you get when you deploy the template:
- An S3 bucket for CodePipeline artifacts (versioned, encrypted, retained on stack deletion)
- An IAM role for CodePipeline (least privilege)
- An IAM role for CodeBuild (least privilege)
- A CodeBuild project with an inline buildspec that:
  - Uses Go 1.23
  - Runs vet and formatting checks
  - Builds binaries for linux/amd64 and windows/amd64
  - Optionally uploads build artifacts to an S3 “release” bucket
- A CodePipeline with two stages: Source (GitHub) and Build (CodeBuild)

Important defaults/assumptions:
- Region: deploy this stack in eu-west-1 (Ireland) as indicated in the template comments.
- Source: GitHub via an existing CodeStar Connection (you supply its ARN).
- Optional S3 upload: provide an S3 bucket name to publish releases; otherwise upload is skipped.

Note about repo paths and binary names:
- The current CLI entry point in this repo is ./cmd/gocomicwriter. The template’s inline buildspec currently builds from ./cmd/gocomic and names artifacts gocomic-*. If your repo uses ./cmd/gocomicwriter (as this one does), adjust those two build lines in the template after deployment or update the template before deploying. See “Adjusting build paths/names” below.


1) Prerequisites
- AWS account with permissions to create IAM roles, S3 buckets, CodeBuild, and CodePipeline
- AWS CLI v2 configured for the target account
- A GitHub repository that contains this code
- A CodeStar Connection to GitHub in eu-west-1
  - In the AWS Console: Developer Tools → Connections → Create connection → GitHub → Follow the OAuth steps
  - When done, copy the Connection ARN; you’ll pass it to the stack as GitHubConnectionArn


2) Parameters you will provide
- ProjectName: logical name for created resources (default gocomicwriter)
- PipelineName: CodePipeline name (default gocomicwriter-pipeline)
- GitHubConnectionArn: ARN of the CodeStar connection you created (required)
- GitHubOwner: GitHub org/user that owns the repo (e.g., alexa)
- GitHubRepo: Repository name (e.g., gocomicwriter)
- BranchName: branch to track (default main)
- ReleaseBucketName: optional S3 bucket to upload artifacts to (leave empty to skip S3 uploads)
- BuildImage: CodeBuild image (default aws/codebuild/standard:7.0)
- ComputeType: CodeBuild size (default BUILD_GENERAL1_SMALL)


3) Deploy the stack (recommended: AWS CLI)
Example using the AWS CLI from the repository root:

PowerShell (Windows):

aws cloudformation deploy `
  --region eu-west-1 `
  --stack-name gocomicwriter-pipeline `
  --template-file docs/aws-codepipeline.yml `
  --capabilities CAPABILITY_NAMED_IAM `
  --parameter-overrides `
    ProjectName=gocomicwriter `
    PipelineName=gocomicwriter-pipeline `
    GitHubConnectionArn=arn:aws:codestar-connections:eu-west-1:<ACCOUNT_ID>:connection/<ID> `
    GitHubOwner=<GITHUB_OWNER> `
    GitHubRepo=gocomicwriter `
    BranchName=main `
    ReleaseBucketName=<optional-s3-bucket-name-or-empty>

Bash (macOS/Linux):

aws cloudformation deploy \
  --region eu-west-1 \
  --stack-name gocomicwriter-pipeline \
  --template-file docs/aws-codepipeline.yml \
  --capabilities CAPABILITY_NAMED_IAM \
  --parameter-overrides \
    ProjectName=gocomicwriter \
    PipelineName=gocomicwriter-pipeline \
    GitHubConnectionArn=arn:aws:codestar-connections:eu-west-1:<ACCOUNT_ID>:connection/<ID> \
    GitHubOwner=<GITHUB_OWNER> \
    GitHubRepo=gocomicwriter \
    BranchName=main \
    ReleaseBucketName=<optional-s3-bucket-name-or-empty>

Notes:
- The stack creates a dedicated S3 bucket for pipeline artifacts (retained on delete). If you later delete the stack, you must empty this bucket manually before deleting it.
- CAPABILITY_NAMED_IAM is required because the stack creates IAM roles.


4) How the pipeline works
- Source stage: Monitors the specified GitHub branch via the CodeStar connection. DetectChanges is enabled; pushes to the branch will start the pipeline automatically.
- Build stage: Triggers the CodeBuild project defined by the stack.
  - Buildspec highlights:
    - install: Go 1.23 runtime
    - pre_build: go version, go mod download, go vet, gofmt check (fails if any file needs formatting)
    - build: cross‑compile for linux/amd64 and windows/amd64 into the dist/ directory
    - post_build: attempts to derive a release tag; if none, falls back to build-<build-number>
      - If ReleaseBucketName is provided, dist/ is uploaded to s3://<bucket>/releases/<tag-or-build>/
  - Artifacts: dist/**/* is emitted to CodePipeline as BuildOutput

Where to find results:
- CodeBuild logs: CloudWatch Logs for the CodeBuild project named <ProjectName>-build
- Pipeline artifacts: the S3 bucket created by the stack (ArtifactStore)
- Optional releases in S3: s3://<ReleaseBucketName>/releases/<tag-or-build>/


5) Adjusting build paths/names (if your CLI path differs)
If your CLI lives at ./cmd/gocomicwriter (this repo) instead of ./cmd/gocomic (what the template’s default buildspec uses), change the two build commands in the inline BuildSpec within the stack to:
- GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o "$ARTIFACT_DIR/gocomicwriter-linux-amd64" ./cmd/gocomicwriter
- GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o "$ARTIFACT_DIR/gocomicwriter-windows-amd64.exe" ./cmd/gocomicwriter

Ways to apply the change:
- Before first deploy: edit docs/aws-codepipeline.yml accordingly and deploy.
- After deploy: update the stack using the same aws cloudformation deploy command with your edited template; CodeBuild will use the new commands for subsequent runs.


6) Optional: preparing an S3 release bucket
If you want CodeBuild to upload artifacts to S3 (post_build step), create a bucket and pass its name as ReleaseBucketName. The stack attaches least‑privilege permissions for CodeBuild to write to that bucket.
- Example bucket name: gocomicwriter-artifacts-<account-id>-eu-west-1
- Objects will be written under releases/<tag-or-build>/
- Public access is not required; you can keep the bucket private and share internally, or front it with CloudFront if you need public distribution.


7) Updating or deleting the pipeline
- Update configuration: edit docs/aws-codepipeline.yml and re‑run the aws cloudformation deploy command with the same stack name.
- Delete: delete the CloudFormation stack. The artifact bucket uses DeletionPolicy: Retain; empty and delete it separately if you want it gone.


8) Troubleshooting
- Source stage stuck in “Retry” or “Not connected”:
  - Ensure the CodeStar Connection is in “Connected” status in eu-west-1 and that the GitHub repo/branch names match the parameters.
- AccessDenied uploading to the release bucket:
  - Confirm you passed ReleaseBucketName and that the bucket exists in the same account/region; the stack adds write permissions to that exact bucket only.
- Build fails on formatting step:
  - Run gofmt -s -w . locally and push again.
- Build cannot find package ./cmd/gocomic:
  - Update the build commands to point to ./cmd/gocomicwriter as described above.
- Go toolchain version:
  - The template requests Go 1.23 on the aws/codebuild/standard:7.0 image. If AWS updates images and the version label changes, adjust the runtime-versions section or BuildImage parameter.


9) Customizing the pipeline
- Add tests: insert go test ./... into the build phase as needed.
- Change compute size or image: use ComputeType and BuildImage parameters.
- Add more stages (e.g., deployment): extend the CodePipeline Stages section in the template.
- Tag‑triggered releases: the buildspec already tries to detect tags; push an annotated tag (e.g., v0.1.0) to create a release folder in S3 when ReleaseBucketName is set.


10) Clean up costs
- CodePipeline and CodeBuild incur small charges when running; S3 storage for artifacts and logs may persist. Clean up by deleting the stack and emptying S3 buckets when no longer needed.


Appendix: File locations
- Template: docs/aws-codepipeline.yml
- CLI entry point (current repo): cmd/gocomicwriter/main.go
- Version string: internal/version/version.go
