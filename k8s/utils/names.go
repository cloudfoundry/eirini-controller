package utils

import (
	"fmt"
	"regexp"
	"strings"

	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	"code.cloudfoundry.org/eirini-controller/util"
	"github.com/pkg/errors"
)

const sanitizedNameMaxLen = 40

func SanitizeName(name, fallback string) string {
	return SanitizeNameWithMaxStringLen(name, fallback, sanitizedNameMaxLen)
}

func SanitizeNameWithMaxStringLen(name, fallback string, maxStringLen int) string {
	validNameRegex := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)
	sanitizedName := strings.ReplaceAll(strings.ToLower(name), "_", "-")

	if validNameRegex.MatchString(sanitizedName) {
		return truncateString(sanitizedName, maxStringLen)
	}

	return truncateString(fallback, maxStringLen)
}

func truncateString(str string, num int) string {
	if len(str) > num {
		return str[0:num]
	}

	return str
}

func GetStatefulsetName(lrp *eiriniv1.LRP) (string, error) {
	nameSuffix, err := util.Hash(fmt.Sprintf("%s-%s", lrp.Spec.GUID, lrp.Spec.Version))
	if err != nil {
		return "", errors.Wrap(err, "failed to generate hash")
	}

	namePrefix := fmt.Sprintf("%s-%s", lrp.Spec.AppName, lrp.Spec.SpaceName)
	namePrefix = SanitizeName(namePrefix, lrp.Spec.GUID)

	return fmt.Sprintf("%s-%s", namePrefix, nameSuffix), nil
}
