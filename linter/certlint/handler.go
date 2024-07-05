package certlint

import (
	"bufio"
	"context"
	"os/exec"

	"github.com/pkimetal/pkimetal/config"
	"github.com/pkimetal/pkimetal/linter"
	"github.com/pkimetal/pkimetal/logger"

	"go.uber.org/zap"
)

type Certlint struct{}

func init() {
	// Register Certlint.
	(&linter.Linter{
		Name:         "certlint",
		Version:      getCertlintVersion(),
		Url:          "https://github.com/certlint/certlint",
		Unsupported:  linter.NonCertificateProfileIDs,
		NumInstances: config.Config.Linter.Certlint.NumProcesses,
		Interface:    func() linter.LinterInterface { return &Certlint{} },
	}).Register()
}

func getCertlintVersion() string {
	// Extract the Certlint version with the help of the Ruby interpreter.
	versionString := linter.NOT_INSTALLED
	cmd := exec.Command("ruby", "-I", "lib:ext", "-e", `#!/usr/bin/ruby -Eutf-8:utf-8
# encoding: UTF-8
require 'certlint'
$stdout.sync = true
$stdout.puts(CertLint::VERSION)
`)
	cmd.Dir = "/usr/local/certlint"
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Logger.Error("cmd.StdoutPipe() failed", zap.Error(err))
	} else if err = cmd.Start(); err != nil {
		logger.Logger.Error("cmd.Start() failed", zap.Error(err))
	} else {
		stdin := bufio.NewScanner(stdout)
		if !stdin.Scan() {
			logger.Logger.Error("stdin.Scan() => false")
		} else {
			versionString = stdin.Text()
		}
	}

	cmd.Wait()
	return versionString
}

func (l *Certlint) StartInstance() (useHandleRequest bool, directory, cmd string, args []string) {
	// Start Certlint server and configure STDIN/STDOUT pipes.
	return false, "/usr/local/certlint", "ruby",
		[]string{"-I", "lib:ext", "-e", `#!/usr/bin/ruby -Eutf-8:utf-8
# encoding: UTF-8
require 'set'
require 'certlint'

$stdout.sync = true

tbr_tevg_profile_ids = Set[` + linter.ProfileIDList(linter.TbrTevgCertificateProfileIDs) + `]
profile_id = -1
pem_cert = ""

begin
	ARGF.each do |line|
		if profile_id == -1
			profile_id = Integer(line)
		else
			pem_cert << line
		end
		if line.include? "END CERTIFICATE"
			m, der_cert = CertLint::PEMLint.lint(pem_cert, 'CERTIFICATE')
			if tbr_tevg_profile_ids.include?(profile_id)
				m << CertLint::CABLint.lint(der_cert)
			else
				m << CertLint.lint(der_cert)
			end
			m << "` + linter.PKIMETAL_ENDOFRESULTS + `"
			$stdout.puts(m)
			profile_id = -1
			pem_cert = ""
		end
	end
rescue Interrupt
end
`}
}

func (l *Certlint) StopInstance(lin *linter.LinterInstance) {
}

func (l *Certlint) HandleRequest(lin *linter.LinterInstance, lreq *linter.LintingRequest, ctx context.Context) []linter.LintingResult {
	// Not used.
	return nil
}
