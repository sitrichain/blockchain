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
package com.rongzer.blockchain.shim;

import static java.nio.charset.StandardCharsets.UTF_8;
import static java.util.stream.Collectors.toList;

import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.zip.GZIPInputStream;

import net.sf.json.JSONObject;

import org.apache.xerces.impl.dv.util.Base64;
import com.rongzer.blockchain.protos.peer.ChaincodeEventPackage.ChaincodeEvent;
import com.rongzer.blockchain.shim.Chaincode.Response;
import com.rongzer.blockchain.shim.impl.Handler;
import com.rongzer.blockchain.shim.ledger.CompositeKey;
import com.rongzer.blockchain.shim.ledger.KeyModification;
import com.rongzer.blockchain.shim.ledger.KeyValue;
import com.rongzer.blockchain.shim.ledger.QueryResultsIterator;

import com.google.protobuf.ByteString;
import com.rongzer.chaincode.utils.JSONUtil;

public interface ChaincodeStub {

	//rongzer,王剑增加
	SecurityContext getSecurityContext();
	/**
	 * Returns the arguments corresponding to the call to
	 * {@link Chaincode#init(ChaincodeStub)} or
	 * {@link Chaincode#invoke(ChaincodeStub)}.
	 *
	 * @return a list of arguments
	 */
	List<byte[]> getArgs();

	/**
	 * Returns the arguments corresponding to the call to
	 * {@link Chaincode#init(ChaincodeStub)} or
	 * {@link Chaincode#invoke(ChaincodeStub)}.
	 *
	 * @return a list of arguments cast to UTF-8 strings
	 */
	List<String> getStringArgs();

	/**
	 * A convenience method that returns the first argument of the chaincode
	 * invocation for use as a function name.
	 *
	 * The bytes of the first argument are decoded as a UTF-8 string.
	 *
	 * @return the function name
	 */
	String getFunction();

	/**
	 * A convenience method that returns all except the first argument of the
	 * chaincode invocation for use as the parameters to the function returned
	 * by #{@link ChaincodeStub#getFunction()}.
	 *
	 * The bytes of the arguments are decoded as a UTF-8 strings and returned as
	 * a list of string parameters..
	 *
	 * @return a list of parameters
	 */
	List<String> getParameters();

	/**
	 * Returns the transaction id
	 *
	 * @return the transaction id
	 */
	String getTxId();

	/**
	 * Invoke another chaincode using the same transaction context.
	 *
	 * @param chaincodeName
	 *            Name of chaincode to be invoked.
	 * @param args
	 *            Arguments to pass on to the called chaincode.
	 * @param channel
	 *            If not specified, the caller's channel is assumed.
	 * @return
	 */
	Response invokeChaincode(String chaincodeName, List<byte[]> args, String channel);

	/**
	 * Returns the byte array value specified by the key, from the ledger.
	 *
	 * @param key
	 *            name of the value
	 * @return value the value read from the ledger
	 */
	byte[] getState(String key);
	
	/**
	 * Returns the byte array value specified by the key, from the ledger.
	 *
	 * @param key
	 *            name of the value
	 * @return value the value read from the ledger
	 */
	List<ByteString> getStates(List<String> keys);

	/**
	 * Writes the specified value and key into the ledger
	 *
	 * @param key
	 *            name of the value
	 * @param value
	 *            the value to write to the ledger
	 */
	void putState(String key, byte[] value);

	/**
	 * Removes the specified key from the ledger
	 *
	 * @param key
	 *            name of the value to be deleted
	 */
	void delState(String key);

	/**
	 * Returns all existing keys, and their values, that are lexicographically
	 * between <code>startkey</code> (inclusive) and the <code>endKey</code>
	 * (exclusive).
	 *
	 * @param startKey
	 * @param endKey
	 * @return an {@link Iterable} of {@link KeyValue}
	 */
	QueryResultsIterator<KeyValue> getStateByRange(String startKey, String endKey);

	/**
	 * Returns all existing keys, and their values, that are prefixed by the
	 * specified partial {@link CompositeKey}.
	 *
	 * If a full composite key is specified, it will not match itself, resulting
	 * in no keys being returned.
	 *
	 * @param compositeKey
	 *            partial composite key
	 * @return an {@link Iterable} of {@link KeyValue}
	 */
	QueryResultsIterator<KeyValue> getStateByPartialCompositeKey(String compositeKey);

	/**
	 * Given a set of attributes, this method combines these attributes to
	 * return a composite key.
	 *
	 * @param objectType
	 * @param attributes
	 * @return a composite key
	 * @throws CompositeKeyFormatException
	 *             if any parameter contains either a U+000000 or U+10FFFF code
	 *             point.
	 */
	CompositeKey createCompositeKey(String objectType, String... attributes);

	/**
	 * Parses a composite key from a string.
	 *
	 * @param compositeKey
	 *            a composite key string
	 * @return a composite key
	 */
	CompositeKey splitCompositeKey(String compositeKey);

	/**
	 * Perform a rich query against the state database.
	 *
	 * @param query
	 *            query string in a syntax supported by the underlying state
	 *            database
	 * @return
	 * @throws UnsupportedOperationException
	 *             if the underlying state database does not support rich
	 *             queries.
	 */
	QueryResultsIterator<KeyValue> getQueryResult(String query);

	/**
	 * Returns the history of the specified key's values across time.
	 *
	 * @param key
	 * @return an {@link Iterable} of {@link KeyModification}
	 */
	QueryResultsIterator<KeyModification> getHistoryForKey(String key);

	/**
	 * Defines the CHAINCODE type event that will be posted to interested
	 * clients when the chaincode's result is committed to the ledger.
	 *
	 * @param name
	 *            Name of event. Cannot be null or empty string.
	 * @param payload
	 *            Optional event payload.
	 */
	void setEvent(String name, byte[] payload);

	/**
	 * Invoke another chaincode using the same transaction context.
	 *
	 * @param chaincodeName
	 *            Name of chaincode to be invoked.
	 * @param args
	 *            Arguments to pass on to the called chaincode.
	 * @return
	 */
	default Response invokeChaincode(String chaincodeName, List<byte[]> args) {
		return invokeChaincode(chaincodeName, args, null);
	}

	/**
	 * rongzer 王剑增加
	 * @param chaincodeName
	 * @param func
	 * @param args
	 * @return
	 */
	default String queryChaincode(String chaincodeName, String func, List<ByteString> args) {
		return queryChaincode(chaincodeName,func,args,null);
	}
	
	/**
	 * rongzer 王剑增加
	 * @param chaincodeName
	 * @param func
	 * @param args
	 * @param channelName
	 * @return
	 */
	default String queryChaincode(String chaincodeName, String func, List<ByteString> args, String channelName) {
		
		List<byte[]> lisArgs = new ArrayList<byte[]>();
		lisArgs.add(func.getBytes());
		for (ByteString bString :args)
		{
			lisArgs.add(bString.toByteArray());
		}
		
		Response response = invokeChaincode(chaincodeName, lisArgs, channelName);
		if (response == null ||  response.getStatus().getCode() != 200)
		{
			return "";
		}
		
		 String strReturn = response.getStringPayload();
			// 自动解压缩处理
		 if (strReturn != null && strReturn.startsWith("{\"__GZIP\":")){
			try {
				JSONObject jGzip = JSONUtil.getJSONObjectFromStr(strReturn);
 
				byte[] bZip = Base64.decode(jGzip.getString("__GZIP"));
				ByteArrayOutputStream out = new ByteArrayOutputStream();
				ByteArrayInputStream in = new ByteArrayInputStream(bZip);

				GZIPInputStream gunzip = new GZIPInputStream(in);
				byte[] buffer = new byte[256];
				int n;
				while ((n = gunzip.read(buffer)) >= 0) {
					out.write(buffer, 0, n);
				}
				return out.toString("UTF-8");
			} catch (IOException e) {
				e.printStackTrace();
			}
				
		 }
		
		return strReturn;
	}
		
	/**
	 * rongzer 王剑增加
	 * @param chaincodeName
	 * @param func
	 * @param args
	 * @return
	 */
	default String invokeChaincode(String chaincodeName, String func, List<ByteString> args) {
		return invokeChaincode(chaincodeName,func,args,null);
	}
	
	/**
	 * rongzer 王剑增加
	 * @param chaincodeName
	 * @param func
	 * @param args
	 * @param channelName
	 * @return
	 */
	default String invokeChaincode(String chaincodeName, String func, List<ByteString> args, String channelName) {
		
		List<byte[]> lisArgs = new ArrayList<byte[]>();
		lisArgs.add(func.getBytes());
		for (ByteString bString :args)
		{
			lisArgs.add(bString.toByteArray());
		}
		
		Response response = invokeChaincode(chaincodeName, lisArgs, channelName);
		if (response == null ||  response.getStatus().getCode() != 200)
		{
			return "";
		}		
		
		return response.getStringPayload();
	}


	/**
	 * Invoke another chaincode using the same transaction context.
	 *
	 * This is a convenience version of
	 * {@link #invokeChaincode(String, List, String)}. The string args will be
	 * encoded into as UTF-8 bytes.
	 *
	 * @param chaincodeName
	 *            Name of chaincode to be invoked.
	 * @param args
	 *            Arguments to pass on to the called chaincode.
	 * @param channel
	 *            If not specified, the caller's channel is assumed.
	 * @return
	 */
	default Response invokeChaincodeWithStringArgs(String chaincodeName, List<String> args, String channel) {
		return invokeChaincode(chaincodeName, args.stream().map(x -> x.getBytes(UTF_8)).collect(toList()), channel);
	}

	/**
	 * Invoke another chaincode using the same transaction context.
	 *
	 * This is a convenience version of {@link #invokeChaincode(String, List)}.
	 * The string args will be encoded into as UTF-8 bytes.
	 *
	 *
	 * @param chaincodeName
	 *            Name of chaincode to be invoked.
	 * @param args
	 *            Arguments to pass on to the called chaincode.
	 * @return
	 */
	default Response invokeChaincodeWithStringArgs(String chaincodeName, List<String> args) {
		return invokeChaincodeWithStringArgs(chaincodeName, args, null);
	}

	/**
	 * Invoke another chaincode using the same transaction context.
	 *
	 * This is a convenience version of {@link #invokeChaincode(String, List)}.
	 * The string args will be encoded into as UTF-8 bytes.
	 *
	 *
	 * @param chaincodeName
	 *            Name of chaincode to be invoked.
	 * @param args
	 *            Arguments to pass on to the called chaincode.
	 * @return
	 */
	default Response invokeChaincodeWithStringArgs(final String chaincodeName, final String... args) {
		return invokeChaincodeWithStringArgs(chaincodeName, Arrays.asList(args), null);
	}

	/**
	 * Returns the byte array value specified by the key and decoded as a UTF-8
	 * encoded string, from the ledger.
	 *
	 * @param key
	 *            name of the value
	 * @return value the value read from the ledger
	 */
	default String getStringState(String key) {
		return new String(getState(key), UTF_8);
	}
	
	/**
	 * Returns the byte array value specified by the key and decoded as a UTF-8
	 * encoded string, from the ledger.
	 *
	 * @param key
	 *            name of the value
	 * @return value the value read from the ledger
	 */
	default List<String> getStringStates(List<String> keys) {
		List<String> lisReturn = new ArrayList<String>();
		List<ByteString> bReturns = getStates(keys);
		for (ByteString bReturn : bReturns){
			lisReturn.add(bReturn.toStringUtf8());
		}
		return lisReturn;
	}

	/**
	 * Writes the specified value and key into the ledger
	 *
	 * @param key
	 *            name of the value
	 * @param value
	 *            the value to write to the ledger
	 */
	default void putStringState(String key, String value) {
		putState(key, value.getBytes(UTF_8));
	}


	/**
	 * rongzer,王剑增加
	 * @param key
	 * @param value
	 */
	default void putState(String key, String value) {

		putStringState(key,value);
	}

	/**
	 * Returns the CHAINCODE type event that will be posted to interested
	 * clients when the chaincode's result is committed to the ledger.
	 *
	 * @return the chaincode event or null
	 */
	ChaincodeEvent getEvent();

	Handler getHandler();
}
