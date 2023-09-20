package com.rongzer.blockchain.shim;

import com.google.protobuf.ByteString;
import com.google.protobuf.Timestamp;

//rongzer,王剑增加
public class SecurityContext{
	
	String channelId = null;
	ByteString callerCert = null;
	ByteString metadata = null;
	Timestamp txTimestamp = null;
	String chainCodeName = null;
	
	public SecurityContext(String channelId,ByteString callerCert, ByteString metadata,Timestamp txTimestamp,String chainCodeName)
	{
		this.channelId = channelId;
		this.callerCert = callerCert;
		this.metadata = metadata;
		this.txTimestamp = txTimestamp;
		this.chainCodeName = chainCodeName;
	}
	
	
	public String getChannelId() {
		return channelId;
	}


	public void setChannelId(String channelId) {
		this.channelId = channelId;
	}


	public ByteString getCallerCert() {
		return callerCert;
	}
	public void setCallerCert(ByteString callerCert) {
		this.callerCert = callerCert;
	}
	public ByteString getMetadata() {
		return metadata;
	}
	public void setMetadata(ByteString metadata) {
		this.metadata = metadata;
	}
	public Timestamp getTxTimestamp() {
		return txTimestamp;
	}
	public void setTxTimestamp(Timestamp txTimestamp) {
		this.txTimestamp = txTimestamp;
	}
	public String getChainCodeName() {
		return chainCodeName;
	}
	public void setChainCodeName(String chainCodeName) {
		this.chainCodeName = chainCodeName;
	}
	
}