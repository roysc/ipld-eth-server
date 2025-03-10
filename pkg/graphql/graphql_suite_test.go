// VulcanizeDB
// Copyright © 2020 Vulcanize

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package graphql_test

import (
	"io/ioutil"
	"testing"

	"github.com/cerc-io/ipld-eth-server/v4/pkg/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestGraphQL(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "graphql test suite")
}

var _ = BeforeSuite(func() {
	log.SetOutput(ioutil.Discard)
})
