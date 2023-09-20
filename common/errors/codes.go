/*
 Copyright Digital Asset Holdings, LLC 2016 All Rights Reserved.
 Copyright IBM Corp. 2017 All Rights Reserved.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package errors

// A set of constants for error reason codes, which is based on HTTP codes
// http://www.iana.org/assignments/http-status-codes/http-status-codes.xhtml
const (
	// Invalid inputs on API calls
	BadRequest = "400"

	// Not Found (eg chaincode not found)
	NotFound = "404"

	// Internal server errors that are not classified below
	Internal = "500"
)

// A set of constants for component codes
const (
	// BCCSP is fabic/BCCSP
	BCCSP = "CSP"
)
