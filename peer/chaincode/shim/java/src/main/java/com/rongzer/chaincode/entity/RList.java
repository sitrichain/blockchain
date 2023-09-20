package com.rongzer.chaincode.entity;

import java.util.ArrayList;
import java.util.List;

import org.apache.log4j.MDC;
import com.rongzer.blockchain.protos.common.Rbc.RListHeight;
import com.rongzer.blockchain.shim.ChaincodeStub;

import com.google.protobuf.ByteString;
import com.rongzer.chaincode.utils.StringUtil;

/**
 * 自动带时间顺序的列表
 * @author Administrator
 *
 */
public class RList {
	//数据操作对象
	private ChaincodeStub stub = null;
	
	//操作key
	private String rListName = "";
	
	private final static String prefix = "__RLIST_";
	private int seq = 0; //起始值为0.

	
	public static RList getInstance(ChaincodeStub stub,String rListName)	
	{
		RList rList = null;
		//增加线程级缓存，可以避免多次初始化
		if (MDC.get(prefix+rListName) != null )
		{
			rList = (RList)MDC.get(prefix+rListName);
		}else
		{
			rList = new RList(stub,rListName);
			MDC.put(prefix+rListName, rList);
		}
		
		return rList;
	}
	
	private RList(ChaincodeStub stub,String rListName)
	{
		this.rListName = rListName;
		this.stub = stub;
	}
	
	private int getSeq()
	{
		return seq++;
	}
	
	
	/**
	 * 增加一个数据
	 * @param id
	 */
	public void add(String id)
	{
		stub.putStringState(prefix+"ADD:"+rListName+","+getSeq(),id);
	}
	
	/**
	 * 在某一个位置增加一个数据
	 * @param nIndex
	 * @param id
	 */
	public void add(int nIndex,String id)
	{
		stub.putStringState(prefix+"ADD:"+rListName+","+getSeq(),nIndex+","+id);
	}
	
	
	/**
	 * 移除一个数据
	 * @param id
	 */
	public void remove(String id)
	{
		stub.putStringState(prefix+"DEL:"+rListName+","+getSeq(),id);
	}
	
	/**
	 * 总长度
	 * @param id
	 */
	public int size()
	{
		return StringUtil.toInt(stub.getStringState(prefix+"LEN:"+rListName),0);
	}
	
	/**
	 * 取某一个位置的值,尝试一次从RList取20个下标的缓存
	 * @param id
	 */
	public String get(int nIndex)
	{
		
		//首先尝试从MDC取，避免遍历时对PEER的频繁访问
		String theKey = prefix+"GET:"+rListName+","+nIndex;
		String cacheKey = "__GET_STATE_"+theKey;
		if (MDC.get(cacheKey) != null){
			ByteString bReturn = (ByteString)MDC.get(cacheKey);
			return bReturn.toStringUtf8();
		}
		
		//循环访问20个
		List<String> keys = new ArrayList<String>();
		for (int i=0;i<20;i++){
			keys.add(prefix+"GET:"+rListName+","+(nIndex-i));
		}
		List<String> lisReturn = stub.getStringStates(keys);
		if (lisReturn.size()>0){
			
			//处理值的预读,处理性能更佳
			List<String> lisData1 = stub.getStringStates(lisReturn);
			
			return lisReturn.get(0);
		}
		
		/**/

		return stub.getStringState(prefix+"GET:"+rListName+","+nIndex);
	}
	
	/**
	 * 总长度
	 * @param id
	 */
	public int indexOf(String id)
	{
		return StringUtil.toInt(stub.getStringState(prefix+"IDX:"+rListName+","+id));
	}
	
	/**
	 * 取某个ID在RList上的块位置
	 * @param id
	 */
	public int getHeightById(String id)
	{
		return StringUtil.toInt(stub.getStringState(prefix+"HGT:"+rListName+","+id));
	}
	
	/**
	 * 取某个ID在RList上的块位置
	 * @param id
	 */
	public RListHeight getIdsByHeight(int height)
	{
		byte[] bytes = stub.getState(prefix+"IDH:"+rListName+","+height);
				RListHeight rListHeight = null;
		if (bytes != null && bytes.length >0) {
			try {
				rListHeight = RListHeight.parseFrom(bytes);
			} catch (Exception e) {
				e.printStackTrace();
			}
		}
		return rListHeight;
	}
	
}
