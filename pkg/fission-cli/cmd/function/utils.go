package function

import (
	"fmt"
	"github.com/fission/fission/pkg/fission-cli/console"
	"strings"
)

func getNamespaceAndName(str string, defaultNamespace string) (namespace string, name string) {
	var secretNamespace string
	var secretName string
	snSplits := strings.Split(str, ".")
	lenSnSplits := len(snSplits)
	if lenSnSplits == 2 {
		secretNamespace = snSplits[0]
		secretName = snSplits[1]
	} else if lenSnSplits == 1 {
		secretNamespace = defaultNamespace
		secretName = str
	} else {
		// warn the secret is not correct
		secretNamespace = defaultNamespace
		secretName = str
		console.Warn(fmt.Sprintf("Secret %s maybe not correct, its namespace will be set %s", secretName, secretNamespace))
	}
	return secretNamespace, secretName
}
