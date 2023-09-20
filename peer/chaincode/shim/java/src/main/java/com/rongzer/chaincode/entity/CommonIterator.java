package com.rongzer.chaincode.entity;

import com.rongzer.blockchain.protos.peer.ChaincodeShim.QueryResponse;
import com.rongzer.blockchain.shim.ChaincodeStub;

public class CommonIterator {
	ChaincodeStub stub;
	String uuid;
	QueryResponse queryResponse;
	int currentLoc = -1;

	public CommonIterator(ChaincodeStub stub)
	{
		this.stub = stub;	
	}

	public ChaincodeStub getStub() {
		return stub;
	}

	public void setStub(ChaincodeStub stub) {
		this.stub = stub;
	}

	public String getUuid() {
		return uuid;
	}

	public void setUuid(String uuid) {
		this.uuid = uuid;
	}

	public QueryResponse getQueryResponse() {
		return queryResponse;
	}

	public void setQueryResponse(QueryResponse queryResponse) {
		this.queryResponse = queryResponse;
	}

	public int getCurrentLoc() {
		return currentLoc;
	}

	public void setCurrentLoc(int currentLoc) {
		this.currentLoc = currentLoc;
	}
	

}
