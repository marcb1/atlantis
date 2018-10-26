package runtime

import (
	"fmt"
	"os"
    "strings"
	"path/filepath"

	"github.com/hashicorp/go-version"
	"github.com/runatlantis/atlantis/server/events/models"
)

type CheckStepRunner struct {
	TerraformExecutor TerraformExec
	DefaultTFVersion  *version.Version
}

func (p *CheckStepRunner) Run(ctx models.ProjectCommandContext, extraArgs []string, path string) (string, error) {
	tfVersion := p.DefaultTFVersion
	if ctx.ProjectConfig != nil && ctx.ProjectConfig.TerraformVersion != nil {
		tfVersion = ctx.ProjectConfig.TerraformVersion
	}

	checkCmd := p.buildCheckCmd(ctx, extraArgs, path)
	ctx.Log.Debug("check command=" + strings.Join(checkCmd, ","))
	return p.TerraformExecutor.RunCommandWithVersion(ctx.Log, filepath.Clean(path), checkCmd, tfVersion, ctx.Workspace)
}

func (p *CheckStepRunner) buildCheckCmd(ctx models.ProjectCommandContext, extraArgs []string, path string) []string {
	tfVars := p.tfVars(ctx)
	planFile := filepath.Join(path, GetPlanFilename(ctx.Workspace, ctx.ProjectConfig))

	// Check if env/{workspace}.tfvars exist and include it. This is a use-case
	// from Hootsuite where Atlantis was first created so we're keeping this as
	// an homage and a favor so they don't need to refactor all their repos.
	// It's also a nice way to structure your repos to reduce duplication.
	var envFileArgs []string
	envFile := filepath.Join(path, "env", ctx.Workspace+".tfvars")
	if _, err := os.Stat(envFile); err == nil {
		envFileArgs = []string{"-var-file", envFile}
	}

	argList := [][]string{
		// NOTE: we need to quote the plan filename because Bitbucket Server can
		// have spaces in its repo owner names.
		{"show", "-input=false", "-no-color" , fmt.Sprintf("%q", planFile)},
		tfVars,
		extraArgs,
		ctx.CommentArgs,
		envFileArgs,
	}

	return p.flatten(argList)
}

// tfVars returns a list of "-var", "key=value" pairs that identify who and which
// repo this command is running for. This can be used for naming the
// session name in AWS which will identify in CloudTrail the source of
// Atlantis API calls.
func (p *CheckStepRunner) tfVars(ctx models.ProjectCommandContext) []string {
	// NOTE: not using maps and looping here because we need to keep the
	// ordering for testing purposes.
	// NOTE: quoting the values because in Bitbucket the owner can have
	// spaces, ex -var atlantis_repo_owner="bitbucket owner".
	return []string{
		"-var",
		fmt.Sprintf("%s=%q", "atlantis_user", ctx.User.Username),
		"-var",
		fmt.Sprintf("%s=%q", "atlantis_repo", ctx.BaseRepo.FullName),
		"-var",
		fmt.Sprintf("%s=%q", "atlantis_repo_name", ctx.BaseRepo.Name),
		"-var",
		fmt.Sprintf("%s=%q", "atlantis_repo_owner", ctx.BaseRepo.Owner),
		"-var",
		fmt.Sprintf("%s=%d", "atlantis_pull_num", ctx.Pull.Num),
	}
}

func (p *CheckStepRunner) flatten(slices [][]string) []string {
	var flattened []string
	for _, v := range slices {
		flattened = append(flattened, v...)
	}
	return flattened
}
