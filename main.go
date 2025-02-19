package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/dvcrn/maskedemail-cli/pkg"
)

type actionType string

const (
	defaultAppname          string = "maskedemail-cli"

	envTokenVarName 		string = "MASKEDEMAIL_TOKEN"
	envAppVarName 			string = "MASKEDEMAIL_APPNAME"
	envAccountIdVarName 	string = "MASKEDEMAIL_ACCOUNTID"

	flagNameToken           string = "token"
	flagNameAccountID       string = "accountid"

	flagNameEmail			string = "email"
	flagNameDomain			string = "domain"
	flagNameDesc			string = "desc"
	flagNameEnabled			string = "enabled"
	flagNameShowDeleted		string = "show-deleted"
	flagNameShowAllFields   string = "all-fields"

	actionTypeUnknown		= ""
	actionTypeCreate        = "create"
	actionTypeSession       = "session"
	actionTypeDisable       = "disable"
	actionTypeEnable        = "enable"
	actionTypeDelete        = "delete"
	actionTypeUpdate        = "update"
	actionTypeList          = "list"
	actionTypeVersion       = "version"

)

// build info values get passed in from makefile via `-ldflags` argument to `go build`
//   they only exist if within a git repo, otherwise use defaults below
// version is based on a git tag "vX.Y.Z" existing
var buildVersion string = "development"
var buildCommit string = "n/a"

// default / highest level flags
var flagAppname = flag.String("appname", os.Getenv(envAppVarName), "the appname to identify the creator (or "+envAppVarName+" env) (default: "+defaultAppname+")")
var flagToken = flag.String(flagNameToken, "", "the token to authenticate with (or "+envTokenVarName+" env)")
var flagAccountID = flag.String(flagNameAccountID, os.Getenv(envAccountIdVarName), "fastmail account id (or "+envAccountIdVarName+" env)")

// flags for list command
var listCmd = flag.NewFlagSet(actionTypeList, flag.ExitOnError)
var flagShowDeleted = listCmd.Bool(flagNameShowDeleted, false, "show deleted masked emails (true|false) (default false)")
var flagShowAllFields = listCmd.Bool(flagNameShowAllFields, false, "show all masked email fields (true|false) (default false)")

// flags for create command
var createCmd = flag.NewFlagSet(actionTypeCreate, flag.ExitOnError)
var flagCreateDomain = createCmd.String(flagNameDomain, "", "domain for the masked email (optional)")
var flagCreateDescription = createCmd.String(flagNameDesc, "", "description for the masked email (optional)")
var flagCreateEnabled = createCmd.Bool(flagNameEnabled, true, "is masked email enabled (true|false)")

// flags for update command
var updateCmd = flag.NewFlagSet(actionTypeUpdate, flag.ExitOnError)
var flagUpdateEmail = updateCmd.String(flagNameEmail, "", "masked email to update (required)")
var flagUpdateDomain = updateCmd.String(flagNameDomain, "", "domain for the masked email (optional, only updated if argument passed)")
var flagUpdateDescription = updateCmd.String(flagNameDesc, "", "description for the masked email (optional, only updated if argument passed)")

var args        []string
var action      actionType = actionTypeUnknown
var commandArg  string
var envToken    string

func isFlagPassed(set flag.FlagSet, name string) bool {
    found := false
    //fmt.Printf("name: %s\n", name)
    set.Visit(func(f *flag.Flag) {
    //	fmt.Printf("f.Name: %s\n", f.Name)
        if f.Name == name {
            found = true
        }
    })
    return found
}

func init() {
	flag.Parse()

	// get all args after the global args
	args = flag.Args()

	// Define initial help message
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Println("Global Flags:")
		flag.PrintDefaults()
		fmt.Println("")
		fmt.Println("Commands:")

		// create
		fmt.Printf("  %s %s [-%s \"<domain>\"] [-%s \"<description>\"] [-%s=true|false (default true)]\n",
					defaultAppname, actionTypeCreate, flagNameDomain, flagNameDesc, flagNameEnabled)

		// list
		fmt.Printf("  %s %s [-%s] [-%s]\n",
					defaultAppname, actionTypeList, flagNameShowDeleted, flagNameShowAllFields)

		// enable
		fmt.Printf("  %s %s <maskedemail>\n",
					defaultAppname, actionTypeEnable)

		// disable
		fmt.Printf("  %s %s <maskedemail>\n",
					defaultAppname, actionTypeDisable)

		// delete
		fmt.Printf("  %s %s <maskedemail>\n",
					defaultAppname, actionTypeDelete)

		// update
		fmt.Printf("  %s %s -%s <maskedemail> [-%s \"<domain>\"] [-%s \"<description>\"]\n",
					defaultAppname, actionTypeUpdate, flagNameEmail, flagNameDomain, flagNameDesc)

		// session
		fmt.Printf("  %s %s\n",
					defaultAppname, actionTypeSession)

		// version
		fmt.Printf("  %s %s\n",
					defaultAppname, actionTypeVersion)
	}

	// Check global arguments:

	// CLI parameter have precedence over ENV variables
	if *flagToken == "" {
		envToken = os.Getenv(envTokenVarName)
		if envToken != "" {
			*flagToken = envToken
		} else {
			flag.Usage()
			os.Exit(1)
		}
	}

	if *flagAppname == "" {
		*flagAppname = defaultAppname
	}


	// determine command/subcommand
	commandArg = ""
	if len(args) > 0 {
		commandArg = strings.ToLower(args[0])
	}

	switch commandArg {

	case actionTypeVersion:
		action = actionTypeVersion

	case actionTypeCreate:
		action = actionTypeCreate

	case actionTypeSession:
		action = actionTypeSession

	case actionTypeDisable:
		action = actionTypeDisable

	case actionTypeEnable:
		action = actionTypeEnable

	case actionTypeDelete:
		action = actionTypeDelete

	case actionTypeList:
		action = actionTypeList

	case actionTypeUpdate:
		action = actionTypeUpdate
	}
}

func main() {

	client := pkg.NewClient(*flagToken, *flagAppname, "35c941ae")

	switch action {

	case actionTypeVersion:
		fmt.Printf("version: %s\n", buildVersion)
		fmt.Printf("commit: %s\n", buildCommit)

	case actionTypeSession:
		session, err := client.Session()
		if err != nil {
			log.Fatalf("fetching session: %v", err)
		}
		var accIDs []string
		for accID := range session.Accounts {
			if *flagAccountID != "" && *flagAccountID != accID {
				continue
			}
			accIDs = append(accIDs, accID)
		}

		primaryAccountID := session.PrimaryAccounts[pkg.MaskedEmailCapabilityURI]
		sort.Slice(
			accIDs,
			func(i, j int) bool {
				if primaryAccountID == accIDs[i] {
					return true
				}
				return accIDs[i] < accIDs[j]
			},
		)
		for _, accID := range accIDs {
			isPrimary := primaryAccountID == accID
			isEnabled := session.AccountHasCapability(accID, pkg.MaskedEmailCapabilityURI)

			fmt.Printf(
				"%s [%s] (primary: %t, enabled: %t)\n",
				session.Accounts[accID].Name,
				accID,
				isPrimary,
				isEnabled,
			)
		}

	case actionTypeCreate:
		// parse command-specific args
		createCmd.Parse(args[1:])

		domain := strings.TrimSpace(*flagCreateDomain)
		description := strings.TrimSpace(*flagCreateDescription)

		session, err := client.Session()
		if err != nil {
			log.Fatalf("initializing session: %v", err)
		}

		createRes, err := client.CreateMaskedEmail(session, *flagAccountID, domain, *flagCreateEnabled, description)
		if err != nil {
			log.Fatalf("error creating masked email: %v", err)
		}

		// success output
		fmt.Println(createRes.Email)

	case actionTypeDisable:
		maskedemail := strings.TrimSpace(args[1])

		if maskedemail == "" {
			log.Fatalln("Usage: disable <maskedemail>")
		}

		session, err := client.Session()
		if err != nil {
			log.Fatalf("initializing session: %v", err)
		}

		_, err = client.DisableMaskedEmail(session, *flagAccountID, maskedemail)
		if err != nil {
			log.Fatalf("error disabling masked email: %v", err)
		}

		// success output
		fmt.Printf("disabled masked email: %s\n", maskedemail)

	case actionTypeEnable:
		maskedemail := strings.TrimSpace(args[1])

		if maskedemail == "" {
			log.Fatalln("Usage: enable <maskedemail>")
		}

		session, err := client.Session()
		if err != nil {
			log.Fatalf("initializing session: %v", err)
		}

		_, err = client.EnableMaskedEmail(session, *flagAccountID, maskedemail)
		if err != nil {
			log.Fatalf("error enabling masked email: %v", err)
		}

		// success output
		fmt.Printf("enabled masked email: %s\n", maskedemail)

	case actionTypeDelete:
		maskedemail := strings.TrimSpace(args[1])

		if maskedemail == "" {
			log.Fatalln("Usage: delete <maskedemail>")
		}

		session, err := client.Session()
		if err != nil {
			log.Fatalf("initializing session: %v", err)
		}

		_, err = client.DeleteMaskedEmail(session, *flagAccountID, maskedemail)
		if err != nil {
			log.Fatalf("error deleting masked email: %v", err)
		}

		// success output
		fmt.Printf("deleted masked email: %s\n", maskedemail)

	case actionTypeList:
		// parse command-specific args
		listCmd.Parse(args[1:])

		session, err := client.Session()
		if err != nil {
			log.Fatalf("initializing session: %v", err)
		}

		maskedEmails, err := client.GetAllMaskedEmails(session, *flagAccountID)
		if err != nil {
			log.Fatalf("err while creating maskedemail: %v", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)

		// display header line
		if *flagShowAllFields {
			fmt.Fprintln(w, "Masked Email\tFor Domain\tDescription\tState\tID\tCreated At\tLast Email At")
		} else {
			fmt.Fprintln(w, "Masked Email\tFor Domain\tDescription\tState")
		}

		// display each masked email
		for _, email := range maskedEmails {
			// skip deleted masked emails unless flag to show is passed
			if email.State == "deleted" && !*flagShowDeleted {
				continue
			}

			// HACK: trim space here is for hack to deal with possible empty strings
			if *flagShowAllFields {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					email.Email,
					strings.TrimSpace(email.Domain),
					strings.TrimSpace(email.Description),
					email.State,
					email.ID,
					email.CreatedAt,
					email.LastMessageAt)
			} else {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					email.Email,
					strings.TrimSpace(email.Domain),
					strings.TrimSpace(email.Description),
					email.State)
			}
		}
		w.Flush()

	case actionTypeUpdate:
		// parse command-specific args
		updateCmd.Parse(args[1:])

		maskedemail := strings.TrimSpace(*flagUpdateEmail)
		domain := strings.TrimSpace(*flagUpdateDomain)
		description := strings.TrimSpace(*flagUpdateDescription)

		// email arg is required
		if !isFlagPassed(*updateCmd, flagNameEmail) || (maskedemail == "") {
			updateCmd.Usage()
			os.Exit(1)
		}

		session, err := client.Session()
		if err != nil {
			log.Fatalf("initializing session: %v", err)
		}

		fields := pkg.NewUpdateFields(isFlagPassed(*updateCmd, flagNameDomain),
									  domain,
									  isFlagPassed(*updateCmd, flagNameDesc),
									  description)

		_, err = client.UpdateInfo(session, *flagAccountID, maskedemail, fields)
		if err != nil {
			log.Fatalf("error updating masked email: %v", err)
		}

		fmt.Printf("updated %s\n", maskedemail)

	default:
		fmt.Println("action not found\n")
		flag.Usage()
		os.Exit(1)
	}
}
