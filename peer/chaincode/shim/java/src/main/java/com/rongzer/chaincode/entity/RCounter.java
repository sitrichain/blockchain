package com.rongzer.chaincode.entity;

import org.apache.log4j.MDC;
import com.rongzer.blockchain.shim.ChaincodeStub;

import com.rongzer.chaincode.utils.StringUtil;

public class RCounter{

	//数据操作对象
	private ChaincodeStub stub = null;
	
	//操作key
	private String counterKey = "";
	
	private final static String prefix = "__RCOUNTER_";
	
	private int seq = 0; //起始值为0.
	
	public static RCounter getInstance(ChaincodeStub stub,String counterKey)
	{
		RCounter rCounter = null;
		//增加线程级缓存，可以避免多次初始化
		if (MDC.get(prefix+counterKey) != null )
		{
			rCounter = (RCounter)MDC.get(prefix+counterKey);
		}else
		{
			rCounter = new RCounter(stub,counterKey);
			MDC.put(prefix+counterKey, rCounter);
		}
		
		return rCounter;
	}
	
	private int getSeq()
	{
		return seq++;
	}
	
	private RCounter(ChaincodeStub stub,String counterKey)
	{
		this.stub = stub;
		this.counterKey = counterKey;
		
		long lValue = StringUtil.tolong(stub.getStringState(this.counterKey),-99999999999999l);
		if (lValue == -99999999999999l)
		{
			stub.putStringState(this.counterKey,"0");
		}
	}
	
	public long value()
	{
		String strValue = stub.getStringState(this.counterKey);
		long lValue = StringUtil.tolong(strValue,-99999999999999l);
		if (lValue == -99999999999999l)
		{
			stub.putStringState(this.counterKey,"0");
			lValue = 0;
		}
		return lValue;
	}
	
	public void set(long lValue)
	{
		stub.putStringState(this.counterKey,""+lValue);
	}
	
	public void add(long lValue)
	{
		stub.putStringState("__CAL_ADD:"+this.counterKey+","+getSeq(),""+lValue);
	}
	
	public void mul(long lValue)
	{
		stub.putStringState("__CAL_MUL:"+this.counterKey+","+getSeq(),""+lValue);
	}
	
	public String currentValue()
	{
		return "##RCOUNT##"+counterKey+"##RCOUNT##";
	}
}
