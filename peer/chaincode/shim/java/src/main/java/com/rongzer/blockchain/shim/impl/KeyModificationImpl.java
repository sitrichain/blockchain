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
package com.rongzer.blockchain.shim.impl;

import java.time.Instant;

import com.rongzer.blockchain.protos.ledger.queryresult.KvQueryResult;
import com.rongzer.blockchain.shim.ledger.KeyModification;

import com.google.protobuf.ByteString;

public class KeyModificationImpl implements KeyModification {

	private final String txId;
	private final ByteString value;
	private final Instant timestamp;
	private final boolean deleted;

	KeyModificationImpl(KvQueryResult.KeyModification km) {
		this.txId = km.getTxId();
		this.value = km.getValue();
		this.timestamp = Instant.ofEpochSecond(km.getTimestamp().getSeconds(), km.getTimestamp().getNanos());
		this.deleted = km.getIsDelete();
	}

	/*
	 * (non-Javadoc)
	 *
	 * @see com.rongzer.blockchain.shim.impl.KeyModification#getTxId()
	 */
	@Override
	public String getTxId() {
		return txId;
	}

	/*
	 * (non-Javadoc)
	 *
	 * @see com.rongzer.blockchain.shim.impl.KeyModification#getValue()
	 */
	@Override
	public byte[] getValue() {
		return value.toByteArray();
	}

	/*
	 * (non-Javadoc)
	 *
	 * @see com.rongzer.blockchain.shim.impl.KeyModification#getStringValue()
	 */
	@Override
	public String getStringValue() {
		return value.toStringUtf8();
	}

	/*
	 * (non-Javadoc)
	 *
	 * @see com.rongzer.blockchain.shim.impl.KeyModification#getTimestamp()
	 */
	@Override
	public Instant getTimestamp() {
		return timestamp;
	}

	/*
	 * (non-Javadoc)
	 *
	 * @see com.rongzer.blockchain.shim.impl.KeyModification#isDeleted()
	 */
	@Override
	public boolean isDeleted() {
		return deleted;
	}

}
