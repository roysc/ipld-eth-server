// VulcanizeDB
// Copyright © 2019 Vulcanize

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

package ipfs_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vulcanize/vulcanizedb/pkg/ipfs/helpers"
	"github.com/vulcanize/vulcanizedb/pkg/ipfs/helpers/mocks"
)

var _ = Describe("Converter", func() {
	Describe("Convert", func() {
		It("Converts StatediffPayloads into IPLDPayloads", func() {
			mockConverter := mocks.PayloadConverter{}
			mockConverter.ReturnIPLDPayload = &helpers.MockIPLDPayload
			ipldPayload, err := mockConverter.Convert(helpers.MockStatediffPayload)
			Expect(err).ToNot(HaveOccurred())
			Expect(ipldPayload).To(Equal(&helpers.MockIPLDPayload))
			Expect(mockConverter.PassedStatediffPayload).To(Equal(helpers.MockStatediffPayload))
		})
	})
})
