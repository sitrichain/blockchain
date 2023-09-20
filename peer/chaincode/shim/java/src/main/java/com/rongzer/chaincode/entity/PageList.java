package com.rongzer.chaincode.entity;

import java.util.ArrayList;
import java.util.Date;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ForkJoinPool;
import java.util.concurrent.RecursiveTask;

import net.sf.json.JSONArray;
import net.sf.json.JSONObject;

import org.apache.commons.codec.binary.Hex;
import org.apache.commons.logging.Log;
import org.apache.commons.logging.LogFactory;
import org.apache.xerces.impl.dv.util.Base64;
import com.rongzer.blockchain.sdk.security.CryptoSuite;
import com.rongzer.blockchain.shim.ChaincodeStub;

import com.google.protobuf.ByteString;
import com.rongzer.chaincode.utils.ChainCodeUtils;
import com.rongzer.chaincode.utils.JSONUtil;
import com.rongzer.chaincode.utils.StringUtil;

public class PageList<V> extends ArrayList<V>{
	private static final Log logger = LogFactory.getLog(PageList.class);

	 public final static int PAGE_ROW = 20;

	int rnum = 0;
	int cpno = 0;
	String message = "";

	public int getRnum() {
		return rnum;
	}

	public void setRnum(int rnum) {
		this.rnum = rnum;
	}

	public int getCpno() {
		return cpno;
	}

	public void setCpno(int cpno) {
		this.cpno = cpno;
	}

	public String getMessage() {
		return message;
	}

	public void setMessage(String message) {
		this.message = message;
	}

	public void fromJSON(JSONObject jData,Class<V> clazz)
	{
		if (jData == null || jData.isEmpty())
		{
			return;
		}
		rnum = StringUtil.toInt((String)jData.get("rnum"),0);
		cpno = StringUtil.toInt((String)jData.get("cpno"),0);
		JSONArray jList = jData.getJSONArray("list");
				
		if (jList != null && jList.size() >0)
		{
			for (int i=0;i<jList.size();i++)
			{
				
				try {
					BaseEntity baseEntity = (BaseEntity)clazz.newInstance();
					baseEntity.fromJSON(jList.getJSONObject(i));
					add((V)baseEntity);
				} catch (Exception e) {
					e.printStackTrace();
				} 
			}
		}	
	}
	
	public JSONObject toJSON()
	{
		JSONObject jData = new JSONObject();
		jData.put("rnum", ""+rnum);
		jData.put("cpno", ""+cpno);
		JSONArray jList = new JSONArray();
		cpno = StringUtil.toInt((String)jData.get("cpno"),0);
		for (int i=0;i<size();i++)
		{
			BaseEntity baseEntity = (BaseEntity)get(i);
			jList.add(baseEntity.toJSON());
		}
		jData.put("list", jList);
		
		return jData;
	}
	
	public byte[] getBytes()
	{
		JSONObject jData = toJSON();
		byte[] bReturn = new byte[0];
		try
		{
			bReturn = jData.toString().getBytes("UTF-8");
		}catch(Exception e)
		{
			
		}
		return bReturn;
	}
	
	/**
	 * 批量加密
	 * @param stub
	 */
	public void enc(ChaincodeStub stub){
		int nSize = this.size();
		if (nSize <1 || !(get(0) instanceof TableDataEntity)){
			return;
		}
		TableDataEntity firstDataEntity = (TableDataEntity)get(0);

		String modelName = firstDataEntity.getModelName();
		String tableName = firstDataEntity.getTableName();
		if (StringUtil.isEmpty(modelName) || StringUtil.isEmpty(tableName)){
			return;
		}
		
		//拼写循环加密
		JSONObject jRequest = new JSONObject();
		for (int i=0;i<nSize;i++){
			if (!(get(i) instanceof TableDataEntity)){
				continue;
			}
			TableDataEntity tableDataEntity = (TableDataEntity)get(i);
		
			JSONObject jData = new JSONObject();
			String mainIdHs = tableDataEntity.getHashString("MAIN_ID");
					
			String pubR = tableDataEntity.getPubR();
			String customerNoHs = tableDataEntity.getHashString("CUSTOMER_NO");

			if (StringUtil.isEmpty(modelName) || StringUtil.isEmpty(tableName)){
				continue;
			}
			if (StringUtil.isEmpty(pubR)||StringUtil.isEmpty(mainIdHs)){
				continue;
			}
			
			jData.put("mainId", mainIdHs);
			jData.put("pubR", pubR);
			jData.put("customerNo", customerNoHs);

			tableDataEntity.loadModel(stub);
			String cols = tableDataEntity.getTabelColNotNull();
			jRequest.put(mainIdHs+"_"+pubR+"_"+customerNoHs + "_" + cols, jData);

		}
		
		if (jRequest.isEmpty()){
			return;
		}
		//从解密中心获取密码
		String pubG = ChainCodeUtils.getPubG();
		List<ByteString> lisArgs = new ArrayList<ByteString>();
		lisArgs.add(ByteString.copyFromUtf8("getMainCryptogram"));
		lisArgs.add(ByteString.copyFromUtf8(modelName));
		lisArgs.add(ByteString.copyFromUtf8(tableName));
		lisArgs.add(ByteString.copyFromUtf8(pubG));

		lisArgs.add(ByteString.copyFromUtf8(jRequest.toString()));
		String strReturn = stub.queryChaincode("rbcmodel", "query", lisArgs);
		
		if (StringUtil.isEmpty(strReturn)){
			return;
		}
		JSONObject jCryptograms = JSONUtil.getJSONObjectFromStr(strReturn);
		if (jCryptograms == null){
			return;
		}
		
		if (jCryptograms.get("__ENCGRAM") != null){
			try {
				String thePubR = (String)jCryptograms.get("pubR");
				byte[] bCryptogram = Base64.decode(jCryptograms.getString("__ENCGRAM"));
				byte[] decCryptogram = ChainCodeUtils.decCryptogram(bCryptogram,CryptoSuite.Factory.getCryptoSuite().bytesToPublicKey(Hex.decodeHex(thePubR.toCharArray())));
				jCryptograms= JSONUtil.getJSONObjectFromStr(new String(decCryptogram));

			} catch (Exception e) {
				e.printStackTrace();
			} 
		}
		
		List<Map<String,Object>> lisMapTableEnc = new ArrayList<Map<String,Object>>();

		for (int i=0;i<nSize;i++){
			
			if (!(get(i) instanceof TableDataEntity)){
				continue;
			}
			TableDataEntity tableDataEntity = (TableDataEntity)get(i);

			String mainIdHs = tableDataEntity.getHashString("MAIN_ID");
			String pubR = tableDataEntity.getPubR();
			String customerNoHs = tableDataEntity.getHashString("CUSTOMER_NO");
			JSONObject jCryptogram = (JSONObject)jCryptograms.get(""+mainIdHs+"_"+pubR+"_"+customerNoHs);
			if (jCryptogram == null){
				continue;
			}
			
            Map<String,Object> mapTableEnc = new HashMap<String,Object>();
            tableDataEntity.loadModel(stub);

            mapTableEnc.put("tableDataEntity", tableDataEntity);
            mapTableEnc.put("stub", stub);
            mapTableEnc.put("jCryptogram", jCryptogram);
            mapTableEnc.put("encType", "enc");
			//tableDataEntity.enc(securityCert, mapTableEnc); 
            lisMapTableEnc.add(mapTableEnc);

			//tableDataEntity.enc(stub, jCryptogram); 
		}
		
        CalCryptogramTask calCryptogramTask = new CalCryptogramTask(0,lisMapTableEnc.size()-1,lisMapTableEnc);
        
        ForkJoinPool pool = new ForkJoinPool();
        pool.invoke(calCryptogramTask);
        pool.shutdown();
	}
	
	/**
	 * 批量解密
	 * @param stub
	 */
	public void dec(ChaincodeStub stub){
		int nSize = this.size();
		if (nSize <1 || !(get(0) instanceof TableDataEntity)){
			return;
		}
		long bTime = (new Date()).getTime();

		TableDataEntity firstDataEntity = (TableDataEntity)get(0);

		String modelName = firstDataEntity.getModelName();
		String tableName = firstDataEntity.getTableName();
		if (StringUtil.isEmpty(modelName) || StringUtil.isEmpty(tableName)){
			return;
		}
		
		//拼写循环加密
		JSONObject jRequest = new JSONObject();
		for (int i=0;i<nSize;i++){
			if (!(get(i) instanceof TableDataEntity)){
				continue;
			}
			TableDataEntity tableDataEntity = (TableDataEntity)get(i);
		
			JSONObject jData = new JSONObject();
			String mainIdHs = tableDataEntity.getHashString("MAIN_ID");
			
			String pubR = tableDataEntity.getPubR();
			String customerNoHs = tableDataEntity.getHashString("CUSTOMER_NO");

			if (StringUtil.isEmpty(modelName) || StringUtil.isEmpty(tableName)){
				continue;
			}
			if (StringUtil.isEmpty(pubR)||StringUtil.isEmpty(mainIdHs)){
				continue;
			}
			
			jData.put("mainId", mainIdHs);
			jData.put("pubR", pubR);
			jData.put("customerNo", customerNoHs);

			tableDataEntity.loadModel(stub);
			String cols = tableDataEntity.getTabelColNotNull();
			if (StringUtil.isNotEmpty(cols)) {
				String colsHash = StringUtil.MD5(cols);
				jData.put("colsHash",colsHash);
				jRequest.put(mainIdHs+"_"+pubR+"_"+customerNoHs + "_" + colsHash, jData);
			} else {
				jRequest.put(mainIdHs+"_"+pubR+"_"+customerNoHs, jData);
			}

		}
		
		if (jRequest.isEmpty()){
			return;
		}
		//从解密中心获取密码
		String pubG = ChainCodeUtils.getPubG();
		List<ByteString> lisArgs = new ArrayList<ByteString>();
		lisArgs.add(ByteString.copyFromUtf8("getMainCryptogram"));
		lisArgs.add(ByteString.copyFromUtf8(modelName));
		lisArgs.add(ByteString.copyFromUtf8(tableName));
		lisArgs.add(ByteString.copyFromUtf8(pubG));

		lisArgs.add(ByteString.copyFromUtf8(jRequest.toString()));
		long pTime1 = (new Date()).getTime();

		String strReturn = stub.queryChaincode("rbcmodel", "query", lisArgs);
		long pTime2 = (new Date()).getTime();
		if (pTime2-bTime>200){
			logger.warn(String.format("PageList dec1 %d record waste time %d", nSize,pTime1-bTime));
			logger.warn(String.format("PageList dec2 %d record waste time %d", nSize,pTime2-pTime1));
		}
		if (StringUtil.isEmpty(strReturn)){
			return;
		}
		JSONObject jCryptograms = JSONUtil.getJSONObjectFromStr(strReturn);
		if (jCryptograms == null){
			return;
		}
				
		if (jCryptograms.get("__ENCGRAM") != null){
			try {
				String thePubR = (String)jCryptograms.get("pubR");
				byte[] bCryptogram = Base64.decode(jCryptograms.getString("__ENCGRAM"));
				byte[] decCryptogram = ChainCodeUtils.decCryptogram(bCryptogram,CryptoSuite.Factory.getCryptoSuite().bytesToPublicKey(Hex.decodeHex(thePubR.toCharArray())));
				jCryptograms= JSONUtil.getJSONObjectFromStr(new String(decCryptogram));

			} catch (Exception e) {
				e.printStackTrace();
			} 
		}
		
		List<Map<String,Object>> lisMapTableEnc = new ArrayList<Map<String,Object>>();

		for (int i=0;i<nSize;i++){
			if (!(get(i) instanceof TableDataEntity)){
				continue;
			}
			TableDataEntity tableDataEntity = (TableDataEntity)get(i);

			String mainIdHs = tableDataEntity.getHashString("MAIN_ID");
			
			String pubR = tableDataEntity.getPubR();
			String customerNoHs = tableDataEntity.getHashString("CUSTOMER_NO");
			tableDataEntity.loadModel(stub);
			String cols = tableDataEntity.getTabelColNotNull();
			JSONObject jCryptogram;
			if (StringUtil.isNotEmpty(cols)) {
				String colsHash = StringUtil.MD5(cols);
				if (jCryptograms.get(""+mainIdHs+"_"+pubR+"_"+customerNoHs+"_"+colsHash) == null || !(jCryptograms.get(""+mainIdHs+"_"+pubR+"_"+customerNoHs+"_"+colsHash) instanceof JSONObject)){
					logger.error(String.format("data %s cryptogram is empty", mainIdHs+"_"+pubR+"_"+customerNoHs));
					continue;
				}
				jCryptogram = (JSONObject)jCryptograms.get(""+mainIdHs+"_"+pubR+"_"+customerNoHs+"_"+colsHash);
			} else {
				if (jCryptograms.get(""+mainIdHs+"_"+pubR+"_"+customerNoHs) == null || !(jCryptograms.get(""+mainIdHs+"_"+pubR+"_"+customerNoHs) instanceof JSONObject)){
					logger.error(String.format("data %s cryptogram is empty", mainIdHs+"_"+pubR+"_"+customerNoHs));
					continue;
				}
				jCryptogram = (JSONObject)jCryptograms.get(""+mainIdHs+"_"+pubR+"_"+customerNoHs);
			}

			if (jCryptogram == null){
				continue;
			}
            Map<String,Object> mapTableEnc = new HashMap<String,Object>();
            tableDataEntity.loadModel(stub);
            mapTableEnc.put("tableDataEntity", tableDataEntity);
            mapTableEnc.put("stub", stub);
            mapTableEnc.put("jCryptogram", jCryptogram);
            mapTableEnc.put("encType", "dec");
            lisMapTableEnc.add(mapTableEnc);
            
			//tableDataEntity.dec(stub, jCryptogram); 
		}
        CalCryptogramTask calCryptogramTask = new CalCryptogramTask(0,lisMapTableEnc.size()-1,lisMapTableEnc);
        
        ForkJoinPool pool = new ForkJoinPool();
        pool.invoke(calCryptogramTask);
        pool.shutdown();
		long eTime = (new Date()).getTime();
		if (eTime-bTime>200){
			logger.warn(String.format("PageList dec3 %d record waste time %d", nSize,eTime-bTime));
		}
	}
		
   private class CalCryptogramTask<V> extends RecursiveTask<V> {

        private int from;
        private int to;
        private List<Map<String,Object>> lisMapTableEnc;



        public CalCryptogramTask(int from, int to, List<Map<String,Object>> lisMapTableEnc) {
            this.from = from;
            this.to = to;
            this.lisMapTableEnc = lisMapTableEnc;
        }


        @Override
        protected V compute(){

            try {
                if (to - from <= 5) {
                    mainCryptogram(from,to,lisMapTableEnc);
                    return null;
                }
                int	 middle = (to + from) / 2;

                CalCryptogramTask<Object> leftJoin = new CalCryptogramTask<Object>(from,
                        middle,lisMapTableEnc);
                CalCryptogramTask<Object> rightJoin = new CalCryptogramTask<Object>(
                        middle + 1, to,lisMapTableEnc);

                invokeAll(leftJoin, rightJoin);

            } catch (Exception e) {
                e.printStackTrace();
            }
            return null;
        }

        private void mainCryptogram(int from, int to, List<Map<String,Object>> lisMapTableEnc) throws Exception {
        	for (int i=from;i<=to;i++){
        		Map<String,Object> mapTableEnc = lisMapTableEnc.get(i);
	        	TableDataEntity tableDataEntity = (TableDataEntity) mapTableEnc.get("tableDataEntity");
	        	ChaincodeStub stub = (ChaincodeStub)mapTableEnc.get("stub");
	            JSONObject jCryptogram = (JSONObject)mapTableEnc.get("jCryptogram");
	            String encType = (String)mapTableEnc.get("encType");
	            if ("dec".equals(encType)){
	            	tableDataEntity.dec(stub, jCryptogram);
	            }else{
	            	tableDataEntity.enc(stub, jCryptogram);

	            }
        	}

        }


    }
}
