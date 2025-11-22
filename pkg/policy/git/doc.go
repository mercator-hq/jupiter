// Package git provides Git repository integration for policy management.
//
// This package enables GitOps workflows by cloning policy repositories,
// watching for changes, and automatically reloading policies when commits
// are pushed. It supports HTTPS and SSH authentication, branch-based
// environments, and safe rollback mechanisms.
//
// # Basic Usage
//
//	cfg := &config.GitPolicyConfig{
//		Repository: "https://github.com/company/policies.git",
//		Branch:     "main",
//		Path:       "policies/",
//	}
//
//	repo, err := git.NewRepository(cfg)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	if err := repo.Clone(context.Background()); err != nil {
//		log.Fatal(err)
//	}
//
// # Change Detection
//
// The watcher monitors the repository for changes and triggers reloads:
//
//	watcher := git.NewWatcher(repo, 30*time.Second, reloadCallback)
//	watcher.Start(context.Background())
//
// # Authentication
//
// Supports multiple authentication methods:
//   - Token-based (HTTPS): GitHub, GitLab, Bitbucket tokens
//   - SSH key-based: Public key authentication
//   - None: Public repositories
//
// # Branch-Based Environments
//
// Use different branches for different environments:
//   - dev branch → Development environment
//   - staging branch → Staging environment
//   - main branch → Production environment
//
// # Rollback
//
// Safely rollback to previous policy versions:
//
//	if err := repo.Rollback(ctx, "a1b2c3d4"); err != nil {
//		log.Fatal(err)
//	}
package git
