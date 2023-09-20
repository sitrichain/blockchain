package com.rongzer.chaincode.entity;

import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Date;
import java.util.List;

import net.sf.json.JSONObject;

import org.apache.commons.logging.Log;
import org.apache.commons.logging.LogFactory;
import org.apache.xerces.impl.dv.util.Base64;
import com.rongzer.blockchain.shim.ChaincodeStub;

import com.rongzer.chaincode.utils.ChainCodeUtils;
import com.rongzer.chaincode.utils.JSONUtil;
import com.rongzer.chaincode.utils.StringUtil;

public class TableDataEntity implements BaseEntity{
	
	private static final Log logger = LogFactory.getLog(TableDataEntity.class);
	
	private JSONObject jObject = null;
	private ModelEntity modelEntity = null;
	private TableEntity tableEntity = null;
	
	private JSONObject jGram = null;

	public TableDataEntity(){
		jObject = new JSONObject();
	}
	
	public TableDataEntity(JSONObject jObject){
		fromJSON(jObject);
	}
	
	@Override
	public void fromJSON(JSONObject jObject) {
		this.jObject = jObject; //排序
	}

	//支持在合约中加密
	public void setPubR(String pubR,String modelName, String tableName){
		if (jObject==null) return;

		jObject.put("__PUB_R", pubR);
		jObject.put("__MODEL_NAME", modelName);
		jObject.put("__TABLE_NAME", tableName);
	}
	
	//读取模型数据
	public void loadModel(ChaincodeStub stub){
		String modelName = getModelName();
		String tableName = getTableName();
	
		if (StringUtil.isEmpty(modelName) || StringUtil.isEmpty(tableName)){
			return;
		}
		
		if (modelEntity == null){
			modelEntity = ChainCodeUtils.getModel(stub, modelName);
		}
		if (tableEntity ==null)
		{
			tableEntity = ChainCodeUtils.getTable(stub, modelName, tableName);
		}
	}

	public String getTabelColNotNull() {

		StringBuilder cols = new StringBuilder();
		for (ColumnEntity columnEntity : tableEntity.getColList()) {

			if (null != getString(columnEntity.getColumnName())) {
				cols.append(columnEntity.getColumnName());
			}
		}

		return cols.toString();
	}

	@Override
	public JSONObject toJSON() {
		String tableDataHS = getString("__HS_TABLE_DATA");
		if (StringUtil.isEmpty(tableDataHS)){//表Hash不存在，重新计算Hash
			//全表数据的Hash
			setValue("__HS_TABLE_DATA", StringUtil.MD5("HASH:"+JSONUtil.getSortedJson(jObject).toString()));
		}
		return jObject;
	}
		
	public String getMainID(){
		return getString("MAIN_ID");
	}
	
	public String getModelName(){
		return getString("__MODEL_NAME");
	}

	public String getTableName(){
		return getString("__TABLE_NAME");
	}
	
	public String getPubR(){
		return getString("__PUB_R");
	}

	public String getTableDataHash(){
		return getString("__HS_TABLE_DATA");
	}

	public boolean equals(JSONObject jData){
		// 合约调用时候，合约参数jData已经经过b.client的处理，已经算出明文的__HS_TABLE_DATA。
		String s= jData.getString("__HS_TABLE_DATA");

		// 成员变量jObject是数据库中的记录。
		return s.equals(getString("__HS_TABLE_DATA"));
	}

	/**
	 * 取整型数
	 * @param colName
	 * @return
	 */
	public int getInt(String colName)
	{
		int nReturn = -1;
		try{
			
			if (jObject.get(colName) instanceof Integer){
				nReturn = jObject.getInt(colName);
			}else{
				nReturn = Integer.parseInt(jObject.getString(colName),-1); //jObject.getString(colName) 能支持任意类型的转换成字符串
			}
			
		}catch(Exception e)
		{
			
		}
		return nReturn;
	}
	
	
	/**
	 * 取字符串
	 * @param colName
	 * @return
	 */
	public String getString(String colName)
	{
		String strReturn = "";
		try{
			if (jObject.get(colName) == null){
				return strReturn;
			}
			strReturn = jObject.getString(colName);
		}catch(Exception e)
		{
			
		}
		return strReturn;
	}
	
	/**
	 * 取Hash值
	 * @param colName
	 * @return
	 */
	public String getHashString(String colName)
	{
		if (jObject.get("__HS_"+colName) != null){
			return jObject.getString("__HS_"+colName);
		}
//		return  getString(colName); //错误，需要获取Hash值
		return StringUtil.MD5("HASH:"+getString(colName));
	}


	/**
	 * 还原Json中的字段中的类型，例如整数类型。
	 * @return
	 */
	public JSONObject restoreFieldTypes(ChaincodeStub stub, String modelName, String tableName) throws NumberFormatException
	{
		//logger.info("restoreFieldTypes()=" + tableName);
		if (jObject == null || jObject==null)
		{
			//日志？？
			return null;
		}

		if (tableEntity ==null)
		{
			tableEntity = ChainCodeUtils.getTable(stub, modelName, tableName);
		}

		for (ColumnEntity columnEntity : tableEntity.getColList()) {
			String colName = columnEntity.getColumnName();
			String type = columnEntity.getColumnType();
			//logger.info("colName=" + colName);
			//logger.info("type=" + type);
			if (jObject.has(colName) && !jObject.getString(colName).isEmpty()) {
				String sjonject=jObject.getString(colName);

				try {
					if (type.equals("number"))
					{
						long intVal = Long.parseLong(sjonject);
						jObject.put(colName, intVal);
					}else if (type.equals("float"))
					{
						float  f = Float.parseFloat(sjonject);
						jObject.put(colName, f);
					}
				}catch (NumberFormatException e)
				{
					String msg = String.format("restoreFieldTypes(tableName=%s,colName=%s,value=%s)",tableName,colName,sjonject);
					logger.error(msg);
					throw new NumberFormatException(msg);
				}
			}
		}
		return jObject;
	}
	
	/**
	 * 设值
	 * @param colName
	 * @return
	 */
	public void setValue(String colName,String value)
	{
		jObject.put(colName, value);
	}

	public void enc(ChaincodeStub stub){
		
		String mainId = this.getMainID();
		if (StringUtil.isEmpty(mainId)){
			return;
		}

		String tableName = "";

		try
		{
			tableName = getTableName();

			
			JSONObject jCryptogram = this.getCryptogram(stub);
			if (jCryptogram == null){
				logger.error(String.format("enc() tableName=%s msg=getCryptogram() is null", tableName));
				return;
			}
			
			enc(stub,jCryptogram);
			
		}catch(Exception e)
		{
			logger.error(String.format("enc() tableName=%s msg=%s", tableName, e.getMessage()));
		}		
	}
	
	
	public void enc(ChaincodeStub stub,JSONObject jCryptogram){
		
		if (jCryptogram == null || jCryptogram.isEmpty()){
			return;
		}
		
		String mainId = this.getMainID();
		if (StringUtil.isEmpty(mainId)){
			return;
		}

		String modelName = "";
		String tableName = "";

		try
		{
			modelName = getModelName();
			tableName = getTableName();
			String pubR = this.getPubR();
			if (StringUtil.isEmpty(modelName) || StringUtil.isEmpty(tableName)){
				return;
			}
			
			if (StringUtil.isEmpty(pubR)){
				return;
			}
			
			if (modelEntity == null){
				modelEntity = ChainCodeUtils.getModel(stub, modelName);
			}

			if (modelEntity == null){
				logger.error(String.format("enc() tableName=%s msg=modelEntity is null", tableName));
				return;
			}

			if (tableEntity ==null)
			{
				tableEntity = ChainCodeUtils.getTable(stub, modelName, tableName);
			}
			
			if (tableEntity ==null)
			{
				logger.error(String.format("enc() tableName=%s msg=tableEntity is null", tableName));
				return;
			}			

			// 对于新增、修改场景，设置INDEXED_DATE为当天日期。
			if (jObject.has("INDEXED_DATE"))
			{
				// 若存在日期类型INDEXED_DATE，填入当前日期，用于运维
				SimpleDateFormat sdf = new SimpleDateFormat("yyyy-MM-dd");
				jObject.put("INDEXED_DATE",sdf.format(new Date()
				));
			}

			List<String> indexCols = new ArrayList<String>();
			//索引字段
			for(IndexEntity indexEntity:tableEntity.getIndexList())
			{
				indexCols.addAll(StringUtil.split(indexEntity.getIndexCols()));
			}
			
			//处理数据的hash值
			JSONObject jHashData = new JSONObject();

			//遍历数据进行加密处理
			for (ColumnEntity columnEntity : tableEntity.getColList()){
				String colName = columnEntity.getColumnName();
				
				String value = getString(colName);

				if (StringUtil.isEmpty(value)|| ChainCodeUtils.isCrypto(value)){
					continue;
				}
				jHashData.put(colName, value);

				//保留字、唯一键、检索键、索引字段保留md5的hash值
				if ("MAIN_ID".equals(colName)||"CUSTOMER_NO".equals(colName)||columnEntity.getColumnUnique() == 1 || columnEntity.getColumnIndex() == 1 || indexCols.indexOf(colName)>=0){
					//唯一键、索引键、字段名保留MD5的hash值
					setValue("__HS_"+colName, StringUtil.MD5("HASH:"+value));
				}				
			}
			
			//全表有效数据的Hash
			setValue("__HS_TABLE_DATA", StringUtil.MD5("HASH:"+jHashData.toString()));
			
			//先处理hash的问题
			if (modelEntity.getDecryptService() == null || modelEntity.getDecryptService().trim().length()<1){
				return;
			}
			jGram = jCryptogram;
		
			JSONObject jData = new JSONObject();
			jData.putAll(jObject);
			//遍历数据进行加密处理
			for (ColumnEntity columnEntity : tableEntity.getColList()){
				String colName = columnEntity.getColumnName();
				String value = getString(colName);

				if (StringUtil.isEmpty(value)){
					continue;
				}
				
				if (StringUtil.isEmpty((String)jCryptogram.get(colName))){
					continue;
				}
				
				//数据己加密
				if (StringUtil.isEmpty(value) || ChainCodeUtils.isCrypto(value)){
					continue;
				}
				try{
					//byte[] cryptogram =Base64.decode(jData.getString(colName)); //错误
					byte[] cryptogram =Base64.decode(jCryptogram.getString(colName));
					
					byte[] bReturn = ChainCodeUtils.enc(cryptogram, value.getBytes());
					jData.put(colName, Base64.encode(bReturn));
				}catch(Exception e1)
				{
					logger.error(String.format("enc() tableName=%s msg=%s", tableName, e1.getMessage()));
				}
			}
			
			jObject = jData;
		}catch(Exception e)
		{
			logger.error(String.format("enc() tableName=%s msg=%s", tableName, e.getMessage()));
		}		
	}

	public void dec(ChaincodeStub stub){
		
		String mainId = this.getMainID();
		if (StringUtil.isEmpty(mainId)){
			logger.error(String.format("dec() msg=mainId is null"));
			return;
		}

		String modelName = "";
		String tableName = "";

		try
		{
			modelName = getModelName();
			tableName = getTableName();
			String pubR = this.getPubR();
			if (StringUtil.isEmpty(modelName) || StringUtil.isEmpty(tableName)){
				logger.error(String.format("dec() tableName=%s msg=%s is null", tableName));
				return;
			}
			
			if (StringUtil.isEmpty(pubR)){
				return;
			}
			
			if (modelEntity == null){
				modelEntity = ChainCodeUtils.getModel(stub, modelName);
			}

			if (modelEntity == null){

				return;
			}
			
			if (modelEntity.getDecryptService() == null || modelEntity.getDecryptService().trim().length()<1){
				return;
			}
			
			if (tableEntity ==null)
			{
				tableEntity = ChainCodeUtils.getTable(stub, modelName, tableName);
			}
			
			if (tableEntity ==null)
			{
				return;
			}

			JSONObject jCryptogram = this.getCryptogram(stub);
			if (jCryptogram == null){

				logger.error(String.format("dec() tableName=%s msg=getCryptogram() is null", tableName));
				return;
			}

			jGram = jCryptogram;
			dec(stub,jCryptogram);
			
		}catch(Exception e)
		{
			logger.error(String.format("dec() tableName=%s msg=%s", tableName, e.getMessage()));
		}
	}
	
	public void dec(ChaincodeStub stub,JSONObject jCryptogram){
		if (jCryptogram == null || jCryptogram.isEmpty()){
			return;
		}
		String modelName = getModelName();
		String tableName = getTableName();
		String pubR = this.getPubR();
		if (StringUtil.isEmpty(modelName) || StringUtil.isEmpty(tableName)){
			return;
		}
		
		if (StringUtil.isEmpty(pubR)){
			return;
		}
		
		if (tableEntity ==null)
		{
			tableEntity = ChainCodeUtils.getTable(stub, modelName, tableName);
		}

		JSONObject jData = new JSONObject();
		jData.putAll(jObject);

		//遍历数据进行加密处理
		for (ColumnEntity columnEntity : tableEntity.getColList()){
			String colName = columnEntity.getColumnName();
			String value = getString(colName);

			if (StringUtil.isEmpty(value)){
				continue;
			}
			
			if (StringUtil.isEmpty((String)jCryptogram.get(colName))){
				continue;
			}
			
			//数据未加密,直接返回
			if (StringUtil.isEmpty(value) || !ChainCodeUtils.isCrypto(value)){
				continue;
			}

			try
			{
				
				byte[] msg = Base64.decode(value);
				if (msg.length>1 && ChainCodeUtils.isCrypto(msg)){
					
					byte[] cryptogram = Base64.decode(jCryptogram.getString(colName));
					
					byte[] bReturn = ChainCodeUtils.dec(cryptogram, msg);

					jData.put(colName, new String(bReturn,"UTF-8"));
				}
			}catch(Exception e1)
			{
				logger.error(String.format("dec() tableName=%s msg=%s", tableName, e1.getMessage()));
			}
		}
		
		jObject = jData;
	}
	
	
	/**
	 * 获取解密密码
	 * @param stub
	 * @return
	 */
	private JSONObject getCryptogram(ChaincodeStub stub)
	{
		
		String mainIdHash = getHashString("MAIN_ID");
		if (StringUtil.isEmpty(mainIdHash)){
			logger.error(String.format("dec() mainId is null"));
			return null;
		}

		try
		{
			String modelName = getModelName();
			String tableName = getTableName();
			String pubR = this.getPubR();
			String customerNoMD5 = getHashString("CUSTOMER_NO");
			loadModel(stub);
			String colHash = getTabelColNotNull();

			if (StringUtil.isEmpty(modelName) || StringUtil.isEmpty(tableName)){
				logger.error(String.format("dec() getCryptogram modelName or tableName is null"));
				return null;
			}
			if (StringUtil.isEmpty(pubR)){
				logger.error(String.format("dec() getCryptogram pubR is null"));
				return null;
			}

			if (StringUtil.isNotEmpty(pubR)){
				return ChainCodeUtils.getCryptogram(stub, modelName, tableName, mainIdHash, customerNoMD5, pubR,colHash);
			}


		}catch(Exception e)
		{
			e.printStackTrace();
			logger.error(String.format("dec() getCryptogram=%s msg=%s", e.getMessage()));
		}
		return null;
	}
	
	//处理附件信息
	public void processAttach(ChaincodeStub stub){
		String modelName = getModelName();
		String tableName = getTableName();
		
		if (modelEntity == null){
			modelEntity = ChainCodeUtils.getModel(stub, modelName);
		}

		if (modelEntity == null){
			return;
		}

		if (tableEntity ==null)
		{
			tableEntity = ChainCodeUtils.getTable(stub, modelName, tableName);
		}
		
		if (tableEntity ==null)
		{
			return;
		}		
		
		for (ColumnEntity colEntity : tableEntity.getColList()){//字段列表
			String colType = colEntity.getColumnType();
			if (!"clob".equals(colType)){
				continue;
			}
			
			String colName = colEntity.getColumnName();
			String value = this.getString(colName);
			String preKey = String.format("%s_%s_%s_",modelName,tableName,colName);

			if (value.startsWith(preKey)){//处理clob字段的值
				String clobValue = ChainCodeUtils.getAttach(stub, value);
				if (StringUtil.isNotEmpty(clobValue)){
					value = clobValue;
					try{
						byte[] msg = Base64.decode(value);
						//如果存在密码，对数据进行解密
						if (jGram != null && jGram.get(colName) != null && msg.length>1 && ChainCodeUtils.isCrypto(value)){
							
							byte[] cryptogram = Base64.decode(jGram.getString(colName));
							
							byte[] bReturn = ChainCodeUtils.dec(cryptogram, msg);

							value = new String(bReturn,"UTF-8");
						}
					}catch(Exception e)
					{
						
					}
					jObject.put(colName, value);
				}
				
			}
		}
	}

	public String getTxId() {
		return (String)jObject.get("txId");
	}

	public void setTxId(String txId) {
		jObject.put("txId", txId);
	}

	public String getTxTime() {
		return (String)jObject.get("txTime");
	}

	public void setTxTime(String txTime) {
		jObject.put("txTime", txTime);
	}

	public String getIdKey() {
		return (String)jObject.get("idKey");
	}

	public void setIdKey(String idKey) {
		jObject.put("idKey", idKey);
	}
}
