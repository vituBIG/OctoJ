package cli

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/OctavoBit/octoj/internal/platform"
	jdkreg "github.com/OctavoBit/octoj/internal/registry"
	"github.com/OctavoBit/octoj/pkg/providers"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	var (
		osFlag   string
		archFlag string
	)

	cmd := &cobra.Command{
		Use:   "search [provider@]<version>",
		Short: "Search available JDK versions across all providers",
		Long: `Search for available JDK versions.

Without a provider prefix, searches ALL providers simultaneously and shows
a combined table so you can compare what is available for a given version.

To narrow results to a specific provider use provider@version or
provider <version> syntax.`,
		Example: `  octoj search 21                    # show JDK 21 from ALL providers
  octoj search temurin 21            # only Temurin JDK 21
  octoj search corretto@17           # only Corretto JDK 17
  octoj search 11 --os linux        # JDK 11 for Linux, all providers
  octoj search                       # list common major versions`,
		RunE: func(cmd *cobra.Command, args []string) error {
			det, err := platform.Detect()
			if err != nil {
				return err
			}
			if osFlag != "" {
				det.OS = osFlag
			}
			if archFlag != "" {
				det.Arch = archFlag
			}
			return runSearch(args, det)
		},
	}

	cmd.Flags().StringVar(&osFlag, "os", "", "target OS (windows, linux, darwin)")
	cmd.Flags().StringVar(&archFlag, "arch", "", "target architecture (x64, arm64)")

	return cmd
}

// providerResult holds results (or error) from a single provider query.
type providerResult struct {
	providerName string
	releases     []providers.JDKRelease
	err          error
}

func runSearch(args []string, det *platform.Info) error {
	registry := jdkreg.New()
	ctx := context.Background()

	if len(args) == 0 {
		fmt.Println("Common JDK major versions:")
		fmt.Println()
		for _, v := range []string{"8", "11", "17", "21"} {
			fmt.Printf("  Java %-4s  →  octoj search %s\n", v, v)
		}
		fmt.Println()
		fmt.Printf("Providers available: %s\n", strings.Join(registry.Names(), ", "))
		fmt.Println()
		fmt.Println("Tip: `octoj search 21` searches all providers at once.")
		return nil
	}

	// Parse args: "temurin 21", "corretto@17", or just "21"
	var targetProvider string // empty = all providers
	var version string

	if len(args) >= 2 {
		// "octoj search temurin 21"
		targetProvider = strings.ToLower(args[0])
		version = args[1]
	} else {
		arg := args[0]
		if strings.Contains(arg, "@") {
			// "corretto@17"
			parts := strings.SplitN(arg, "@", 2)
			targetProvider = strings.ToLower(parts[0])
			version = parts[1]
		} else {
			// just a version → search ALL providers
			version = arg
		}
	}

	log.Debug().
		Str("provider", targetProvider).
		Str("version", version).
		Str("os", det.OS).
		Str("arch", det.Arch).
		Msg("searching JDK releases")

	var providerList []providers.Provider

	if targetProvider != "" {
		// Single provider requested
		p, err := registry.Get(targetProvider)
		if err != nil {
			return fmt.Errorf("unknown provider %q — available: %s", targetProvider, strings.Join(registry.Names(), ", "))
		}
		providerList = []providers.Provider{p}
	} else {
		// All providers
		providerList = registry.All()
	}

	// Query all selected providers concurrently
	resultsCh := make(chan providerResult, len(providerList))
	var wg sync.WaitGroup

	for _, p := range providerList {
		wg.Add(1)
		go func(p providers.Provider) {
			defer wg.Done()
			releases, err := p.Search(ctx, version, det.OS, det.Arch)
			resultsCh <- providerResult{providerName: p.Name(), releases: releases, err: err}
		}(p)
	}

	wg.Wait()
	close(resultsCh)

	// Collect results
	var allReleases []providers.JDKRelease
	var errors []string

	for res := range resultsCh {
		if res.err != nil {
			log.Debug().Err(res.err).Str("provider", res.providerName).Msg("provider search error")
			errors = append(errors, fmt.Sprintf("%s: %v", res.providerName, res.err))
			continue
		}
		allReleases = append(allReleases, res.releases...)
	}

	if len(allReleases) == 0 {
		fmt.Printf("No releases found for JDK %s on %s/%s\n", version, det.OS, det.Arch)
		if len(errors) > 0 {
			fmt.Println("\nErrors encountered:")
			for _, e := range errors {
				fmt.Printf("  • %s\n", e)
			}
		}
		return nil
	}

	title := fmt.Sprintf("JDK %s releases available for %s/%s", version, det.OS, det.Arch)
	if targetProvider != "" {
		title = fmt.Sprintf("%s JDK %s releases for %s/%s", targetProvider, version, det.OS, det.Arch)
	}
	fmt.Println(title)
	fmt.Println()
	fmt.Printf("  %-12s  %-22s  %s\n", "PROVIDER", "FULL VERSION", "INSTALL COMMAND")
	fmt.Printf("  %-12s  %-22s  %s\n", strings.Repeat("-", 12), strings.Repeat("-", 22), strings.Repeat("-", 30))

	for _, r := range allReleases {
		installCmd := fmt.Sprintf("octoj install %s@%s", r.Provider, r.Version)
		fmt.Printf("  %-12s  %-22s  %s\n", r.Provider, r.FullVersion, installCmd)
	}

	fmt.Println()
	if len(errors) > 0 {
		fmt.Printf("  (some providers could not be reached: %s)\n", strings.Join(errors, "; "))
	}

	return nil
}
