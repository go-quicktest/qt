// Licensed under the MIT license, see LICENSE file for details.

package qt

var Prefixf = prefixf

// SetVerbosity mutates the checker c to set its verbosity
// if it supports that.
func SetVerbosity(c any, v bool) {
	if c, ok := c.(interface{ setVerbose(bool) }); ok {
		c.setVerbose(v)
	}
}
