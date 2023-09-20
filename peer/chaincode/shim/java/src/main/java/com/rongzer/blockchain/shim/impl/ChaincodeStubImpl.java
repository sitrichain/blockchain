/*
Copyright DTCC, IBM 2016, 2017 All Rights Reserved.

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

import static java.util.stream.Collectors.toList;

import java.util.ArrayList;
import java.util.Collections;
import java.util.Date;
import java.util.List;
import java.util.function.Function;
import java.util.stream.Collectors;

import org.apache.log4j.MDC;
import com.rongzer.blockchain.protos.ledger.queryresult.KvQueryResult;
import com.rongzer.blockchain.protos.ledger.queryresult.KvQueryResult.KV;
import com.rongzer.blockchain.protos.peer.ChaincodeEventPackage.ChaincodeEvent;
import com.rongzer.blockchain.protos.peer.ChaincodeShim.QueryResultBytes;
import com.rongzer.blockchain.shim.Chaincode.Response;
import com.rongzer.blockchain.shim.ChaincodeStub;
import com.rongzer.blockchain.shim.SecurityContext;
import com.rongzer.blockchain.shim.ledger.CompositeKey;
import com.rongzer.blockchain.shim.ledger.KeyModification;
import com.rongzer.blockchain.shim.ledger.KeyValue;
import com.rongzer.blockchain.shim.ledger.QueryResultsIterator;

import com.google.protobuf.ByteString;
import com.google.protobuf.InvalidProtocolBufferException;

class ChaincodeStubImpl implements ChaincodeStub {

	private final String txId;
	private final Handler handler;
	private final List<ByteString> args;
	private ChaincodeEvent event;
	private final SecurityContext securityContext;

	ChaincodeStubImpl(String txId, Handler handler, List<ByteString> args,SecurityContext securityContext) {
		this.txId = txId;
		this.handler = handler;
		this.args = Collections.unmodifiableList(args);
		this.securityContext = securityContext;
	}
	
	@Override
	public SecurityContext getSecurityContext() {
		return securityContext;
	}

	@Override
	public List<byte[]> getArgs() {
		return args.stream().map(x -> x.toByteArray()).collect(Collectors.toList());
	}

	@Override
	public List<String> getStringArgs() {
		return args.stream().map(x -> x.toStringUtf8()).collect(Collectors.toList());
	}

	@Override
	public String getFunction() {
		return getStringArgs().size() > 0 ? getStringArgs().get(0) : null;
	}

	@Override
	public List<String> getParameters() {
		return getStringArgs().stream().skip(1).collect(toList());
	}

	@Override
	public void setEvent(String name, byte[] payload) {
		if (name == null || name.trim().length() == 0) throw new IllegalArgumentException("Event name cannot be null or empty string.");
		if (payload != null) {
			this.event = ChaincodeEvent.newBuilder()
					.setEventName(name)
					.setPayload(ByteString.copyFrom(payload))
					.build();
		} else {
			this.event = ChaincodeEvent.newBuilder()
					.setEventName(name)
					.build();
		}
	}

	@Override
	public ChaincodeEvent getEvent() {
		return event;
	}

	@Override
	public String getTxId() {
		return txId;
	}

	@Override
	public byte[] getState(String key) {
 
		String cacheKey = "__GET_STATE_"+key;
		ByteString bValue = (ByteString)MDC.get(cacheKey);
		if (bValue == null){
			Long readNum = (Long)MDC.get("readNum");
			Long readTime = (Long)MDC.get("readTime");
			//王剑增加，处理智能合约约单次执行中的重复读
			long sTime = (new Date()).getTime();
			bValue = handler.getState(txId, key);
			long eTime = (new Date()).getTime();
			if (readNum != null){
				readNum ++;
				MDC.put("readNum", readNum);
			}
			
			if (readTime != null){
				readTime += (eTime-sTime);
				MDC.put("readTime", readTime);
			}

		}
		if (bValue == null){
			bValue = ByteString.EMPTY;
		}
		
		if (bValue != null){
			MDC.put(cacheKey, bValue);
		}
		
		return bValue.toByteArray();
	}

	@SuppressWarnings("unchecked")
	@Override
	public List<ByteString> getStates(List<String> keys) {
		
		String cacheKey = "__GET_STATES_";
		for (String key : keys){
			cacheKey +=key+",";
		}
		//缓存整体值
		List<ByteString> bReturn = (List<ByteString>)MDC.get(cacheKey);
		if (bReturn == null){
			Long readNum = (Long)MDC.get("readNum");
			Long readTime = (Long)MDC.get("readTime");
			//王剑增加，处理智能合约约单次执行中的重复读
			long sTime = (new Date()).getTime();
			bReturn = handler.getStates(txId, keys);
			long eTime = (new Date()).getTime();
			if (readNum != null){
				readNum ++;
				MDC.put("readNum", readNum);
			}
			
			if (readTime != null){
				readTime += (eTime-sTime);
				MDC.put("readTime", readTime);
			}
			//缓存单个值
			if (keys.size() == bReturn.size()){
				for(int i=0;i<keys.size();i++){
					String cacheKey1 = "__GET_STATE_"+keys.get(i);
					MDC.put(cacheKey1, bReturn.get(i));
				}
			}
		}
		if (bReturn == null){
			bReturn = new ArrayList<ByteString>();
		}
		
		if (bReturn != null){
			MDC.put(cacheKey, bReturn);
		}
		
		return bReturn;
	}
	
	@Override
	public void putState(String key, byte[] value) {
		handler.putState(txId, key, ByteString.copyFrom(value));
	}

	@Override
	public void delState(String key) {
		handler.deleteState(txId, key);
	}

	@Override
	public QueryResultsIterator<KeyValue> getStateByRange(String startKey, String endKey) {
		return new QueryResultsIteratorImpl<KeyValue>(this.handler, getTxId(),
				handler.getStateByRange(getTxId(), startKey, endKey),
				queryResultBytesToKv.andThen(KeyValueImpl::new)
				);
	}

	private Function<QueryResultBytes, KV> queryResultBytesToKv = new Function<QueryResultBytes, KV>() {
		public KV apply(QueryResultBytes queryResultBytes) {
			try {
				return KV.parseFrom(queryResultBytes.getResultBytes());
			} catch (InvalidProtocolBufferException e) {
				throw new RuntimeException(e);
			}
		};
	};

	@Override
	public QueryResultsIterator<KeyValue> getStateByPartialCompositeKey(String compositeKey) {
		return getStateByRange(compositeKey, compositeKey + "\udbff\udfff");
	}

	@Override
	public CompositeKey createCompositeKey(String objectType, String... attributes) {
		return new CompositeKey(objectType, attributes);
	}

	@Override
	public CompositeKey splitCompositeKey(String compositeKey) {
		return CompositeKey.parseCompositeKey(compositeKey);
	}

	@Override
	public QueryResultsIterator<KeyValue> getQueryResult(String query) {
		return new QueryResultsIteratorImpl<KeyValue>(this.handler, getTxId(),
				handler.getQueryResult(getTxId(), query),
				queryResultBytesToKv.andThen(KeyValueImpl::new)
				);
	}

	@Override
	public QueryResultsIterator<KeyModification> getHistoryForKey(String key) {
		return new QueryResultsIteratorImpl<KeyModification>(this.handler, getTxId(),
				handler.getHistoryForKey(getTxId(), key),
				queryResultBytesToKeyModification.andThen(KeyModificationImpl::new)
				);
	}

	private Function<QueryResultBytes, KvQueryResult.KeyModification> queryResultBytesToKeyModification = new Function<QueryResultBytes, KvQueryResult.KeyModification>() {
		public KvQueryResult.KeyModification apply(QueryResultBytes queryResultBytes) {
			try {
				return KvQueryResult.KeyModification.parseFrom(queryResultBytes.getResultBytes());
			} catch (InvalidProtocolBufferException e) {
				throw new RuntimeException(e);
			}
		};
	};

	@Override
	public Response invokeChaincode(final String chaincodeName, final List<byte[]> args, final String channel) {
		// internally we handle chaincode name as a composite name
		final String compositeName;
		if (channel != null && channel.trim().length() > 0) {
			compositeName = chaincodeName + "/" + channel;
		} else {
			compositeName = chaincodeName;
		}
		return handler.invokeChaincode(this.txId, compositeName, args);
	}

	public Handler getHandler(){
		return handler;
	}
}
