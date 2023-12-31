/*
Copyright IBM 2017 All Rights Reserved.

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
package com.rongzer.blockchain.shim.ledger;

/**
 * Query Result associating a state key with a value.
 *
 */
public interface KeyValue {

	/**
	 * Returns the state key.
	 * 
	 * @return
	 */
	String getKey();

	/**
	 * Returns the state value.
	 * 
	 * @return
	 */
	byte[] getValue();

	/**
	 * Returns the state value, decoded as a UTF-8 string.
	 * 
	 * @return
	 */
	String getStringValue();

}