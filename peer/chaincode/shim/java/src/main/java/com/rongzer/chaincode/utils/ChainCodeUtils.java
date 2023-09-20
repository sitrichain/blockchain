package com.rongzer.chaincode.utils;

import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.ObjectInputStream;
import java.io.ObjectOutputStream;
import java.security.KeyPair;
import java.security.MessageDigest;
import java.security.interfaces.ECPrivateKey;
import java.security.interfaces.ECPublicKey;
import java.util.ArrayList;
import java.util.Date;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import net.sf.json.JSONObject;

import org.apache.commons.codec.binary.Hex;
import org.apache.commons.logging.Log;
import org.apache.commons.logging.LogFactory;
import org.apache.log4j.MDC;
import org.apache.xerces.impl.dv.util.Base64;
import com.rongzer.blockchain.sdk.security.CryptoSuite;
import com.rongzer.blockchain.shim.ChaincodeStub;

import com.google.protobuf.ByteString;
import com.rongzer.blockchain.common.RBCRuntimeException;
import com.rongzer.chaincode.entity.ColumnEntity;
import com.rongzer.chaincode.entity.CustomerEntity;
import com.rongzer.chaincode.entity.MessageEntity;
import com.rongzer.chaincode.entity.ModelEntity;
import com.rongzer.chaincode.entity.PageList;
import com.rongzer.chaincode.entity.RoleEntity;
import com.rongzer.chaincode.entity.TableEntity;

public class ChainCodeUtils {
    private static final Log logger = LogFactory.getLog(ChainCodeUtils.class);

    //获取加密类型 ecdh/sm2,非ecdh就是国密
    public static String securityType = null;
    public static String getSecurityType(){
    	return securityType;
    }
	/**
	 * 获取交易时间
	 * @param stub
	 * @return
	 */
	public static String getTxTime(ChaincodeStub stub)
	{
		String strReturn = "";
		long lTime = stub.getSecurityContext().getTxTimestamp().getSeconds()*1000;
		strReturn = StringUtil.dateTimeToStr(new Date(lTime));
		return strReturn;
	}
	
	/**
	 * 根据条件查询用户
	 * @param stub
	 * @param jData
	 * @return
	 */
	public static CustomerEntity queryCustomer(ChaincodeStub stub, String param)
	{
		String cacheKey = "__CUSTOMER_"+param;
		if (MDC.get(cacheKey) != null){
			return (CustomerEntity) MDC.get(cacheKey);
		}
		List<ByteString> lisArgs = new ArrayList<ByteString>();
		lisArgs.add(ByteString.copyFromUtf8("queryOne"));
		lisArgs.add(ByteString.copyFromUtf8(param));
		String strReturn = stub.queryChaincode("rbccustomer","query", lisArgs);
		
		JSONObject jReturn = JSONUtil.getJSONObjectFromStr(strReturn);
		if(jReturn == null || jReturn.isEmpty()){
			return null;
		}
		CustomerEntity customerBean = new CustomerEntity();
		customerBean.fromJSON(jReturn);
		if (customerBean != null){
			MDC.put(cacheKey, customerBean);
		}
		return customerBean;
	}
	
	/**
	 * 根据用户ID查询用户
	 * @param stub
	 * @param customerId
	 * @return
	 */
	public static CustomerEntity queryCustomerByID(ChaincodeStub stub, String customerId)
	{

		return queryCustomer(stub,customerId);
	}
	
	/**
	 * 根据用户NO查询用户
	 * @param stub
	 * @param customerNo
	 * @return
	 */
	public static CustomerEntity queryCustomerByNo(ChaincodeStub stub, String customerNo)
	{
		return queryCustomer(stub,customerNo);
	}
	
	/**
	 * 根据签名公钥查询用户
	 * @param stub
	 * @param searchSign
	 * @return
	 */
	public static CustomerEntity queryExecCustomer(ChaincodeStub stub)
	{

		return queryCustomer(stub, StringUtil.MD5(stub.getSecurityContext().getCallerCert().toStringUtf8().trim()));
	}
	
	
	/**
	 * 根据签名公钥查询用户
	 * @param stub
	 * @param searchSign
	 * @return
	 */
	public static CustomerEntity queryCustomerByCert(ChaincodeStub stub,String searchSign)
	{
		return queryCustomer(stub,StringUtil.MD5(searchSign));
	}
	
	/**
	 * 分页查询所有用户
	 * @param stub
	 * @param nPageNum
	 * @return
	 */
	public static PageList<CustomerEntity> queryAllCustomer(ChaincodeStub stub,String customerType,String customerStatus,int cpno)
	{
		
		JSONObject jData = new JSONObject();
		jData.put("cpno", ""+cpno);
		if (StringUtil.isNotEmpty(customerType))
		{
			jData.put("customerType", ""+customerType);
		}
		if (StringUtil.isNotEmpty(customerType))
		{
			jData.put("customerStatus", ""+customerStatus);
		}
		List<ByteString> lisArgs = new ArrayList<ByteString>();
		lisArgs.add(ByteString.copyFromUtf8("queryAll"));
		lisArgs.add(ByteString.copyFromUtf8(jData.toString()));
		String strReturn  = stub.queryChaincode("rbccustomer","query", lisArgs);
		
		JSONObject jReturn = JSONUtil.getJSONObjectFromStr(strReturn);
		
		PageList<CustomerEntity> listCustomer = new PageList<CustomerEntity>();
		
		listCustomer.fromJSON(jReturn, CustomerEntity.class);

		return listCustomer;
	}
	
	
	/**
	 * 查询所有角色
	 * @param stub
	 * @return
	 */
	@SuppressWarnings("unchecked")
	public static PageList<RoleEntity> getAllRole(ChaincodeStub stub)
	{
		String cacheKey = "__ROLE_";
		if (MDC.get(cacheKey) != null){
			return (PageList<RoleEntity>) MDC.get(cacheKey);
		}
		
		List<ByteString> lisArgs = new ArrayList<ByteString>();
		String strReturn  = stub.queryChaincode("rbccustomer","getAllRole", lisArgs);
		
		JSONObject jReturn = JSONUtil.getJSONObjectFromStr(strReturn);
		
		PageList<RoleEntity> lisRole = new PageList<RoleEntity>();
		
		lisRole.fromJSON(jReturn, RoleEntity.class);
		if (lisRole != null){
			MDC.put(cacheKey,lisRole);
		}

		return lisRole;
	}
	
	/**
	 * 查询应用模型
	 * @param stub
	 * @param modelName
	 * @return 应用模型对象
	 */
	public static ModelEntity getModel(ChaincodeStub stub, String modelName)
	{
		if (StringUtil.isEmpty(modelName)) {
			return null;
		}
		
		String cacheKey = "__MODEL_"+modelName;
		if (MDC.get(cacheKey) != null){
			return (ModelEntity) MDC.get(cacheKey);
		}

		ModelEntity modelEntity = null;
		try {
			List<ByteString> lisArgs = new ArrayList<ByteString>();
			lisArgs.add(ByteString.copyFromUtf8("getModel"));
			lisArgs.add(ByteString.copyFromUtf8(modelName));
			String strReturn = stub.queryChaincode("rbcmodel", "query", lisArgs);
			JSONObject jReturn = JSONUtil.getJSONObjectFromStr(strReturn);
			if(jReturn == null || jReturn.isEmpty()){
				return null;
			}

			modelEntity = new ModelEntity(jReturn);
		} catch (Exception e) {
			logger.error(e.toString());
			return null;
		}
		
		if (modelEntity != null){
			MDC.put(cacheKey,modelEntity);
		}

		return modelEntity;

	}
	
	/**
	 * 获取模型表
	 * @param stub
	 * @param modelName
	 * @param tableName
	 * @return 表对象
	 */
	public static TableEntity getTable(ChaincodeStub stub,  String modelName,String tableName)
	{
		TableEntity tableEntity = null;
		String cacheKey = "__TABLE_"+tableName;
		if (MDC.get(cacheKey) != null){
			return (TableEntity) MDC.get(cacheKey);
		}
		try{
			//处理读写权限
			CustomerEntity customerEntity = queryExecCustomer(stub);
			if (customerEntity == null){
				return null;
			}
			List<ByteString> lisArgs = new ArrayList<ByteString>();
			lisArgs.add(ByteString.copyFromUtf8("getTable"));
			lisArgs.add(ByteString.copyFromUtf8(modelName));
			lisArgs.add(ByteString.copyFromUtf8(tableName));
			String strReturn = stub.queryChaincode("rbcmodel", "query", lisArgs);
			JSONObject jReturn = JSONUtil.getJSONObjectFromStr(strReturn);
			if(jReturn == null || jReturn.isEmpty()){
				return null;
			}

			tableEntity = new TableEntity(jReturn);
			
			List<String> lisRoleNos = StringUtil.split(StringUtil.safeTrim(customerEntity.getRoleNos()));
			//设置字段的读写权限
			for (ColumnEntity columnEntity : tableEntity.getColList()){
				List<String> lisReadRoles =  StringUtil.split(StringUtil.safeTrim(columnEntity.getReadRoles()));
				List<String> lisWriteRoles =  StringUtil.split(StringUtil.safeTrim(columnEntity.getWriteRoles()));
				//如果未设置角色，角色则存在所有权限
				boolean bRead = false;
				boolean bWrite = false;
				if (lisRoleNos.size() <1){
					bRead = true;
					bWrite = true;
				}
				
				if (lisReadRoles.size()<1){
					bRead = true;
				}
				
				if (lisWriteRoles.size()<1){
					bRead = true;
					bWrite = true;
				}
				
				for (String roleNo : lisReadRoles){
					if (lisRoleNos.indexOf(roleNo)>=0){
						bRead = true;
						break;
					}
				}

				for (String roleNo : lisWriteRoles){
					if (lisRoleNos.indexOf(roleNo)>=0){
						bRead = true;
						bWrite = true;
						break;
					}
				}
				columnEntity.setCanRead(bRead);
				columnEntity.setCanWrite(bWrite);
				
			}
			
		}catch(Exception e)
		{
			logger.error(e.toString());
			return null;
		}
		
		if (tableEntity != null){
			MDC.put(cacheKey,tableEntity);
		}
		
		return tableEntity;
	}

	/**
	 * 发送消息通知
	 * @param stub
	 * @param bizId 消息数据ID
	 * @param msgType 消息类型
	 * @param toCustomer 接收会员
	 * @param msgParams	消息参数
	 * @param msgDesc 消息描述
	 */
	public static void sendMessage(ChaincodeStub stub,String bizId,String msgType,String toCustomer){
		List<String> msgParams = new ArrayList<String>();
		msgParams.add(bizId);
		sendMessage(stub,bizId,msgType,toCustomer,msgParams,"");
	}
	
	/**
	 * 发送消息通知
	 * @param stub
	 * @param bizId 消息数据ID
	 * @param msgType 消息类型
	 * @param toCustomer 接收会员
	 * @param msgParams	消息参数
	 * @param msgDesc 消息描述
	 */
	public static void sendMessage(ChaincodeStub stub,String bizId,String msgType,String toCustomer,List<String> msgParams,String msgDesc){
		sendMessage(stub,bizId,msgType,toCustomer,msgParams,msgDesc,"");
	}
	
	/**
	 * 发送消息通知
	 * @param stub
	 * @param bizId 消息数据ID
	 * @param msgType 消息类型
	 * @param toCustomer 接收会员
	 * @param msgParams	消息参数
	 * @param msgDesc 消息描述
	 */
	public static void sendMessage(ChaincodeStub stub,String bizId,String msgType,String toCustomer,List<String> msgParams,String msgDesc,String notifyTime){
		if (StringUtil.isEmpty(bizId)||StringUtil.isEmpty(msgType)||StringUtil.isEmpty(toCustomer)||msgParams ==null || msgParams.size()<1){
			logger.error("sendmessage error:params is error");
			return;
		}
		
		CustomerEntity customerEntity = queryCustomerByNo(stub, toCustomer);
		if (customerEntity == null || StringUtil.isEmpty(customerEntity.getCustomerId())){
			logger.error("sendmessage error:params is empty");
			return;
		}
		
		MessageEntity messageEntity = new MessageEntity();
		messageEntity.setBizId(bizId);
		messageEntity.setMsgType(msgType);
		messageEntity.setToCustomer(toCustomer);
		messageEntity.setMsgParams(msgParams);
		messageEntity.setMsgDesc(msgDesc);
		messageEntity.setReadStatus("0");
		messageEntity.setHandleStatus("0");
		messageEntity.setNotifyTime(notifyTime);
		
		List<ByteString> lisArgs = new ArrayList<ByteString>();
		lisArgs.add(ByteString.copyFromUtf8("sendMessage"));
		lisArgs.add(ByteString.copyFromUtf8(messageEntity.toJSON().toString()));
		//异步发送消息
		stub.invokeChaincode("rbcnotify", "invoke", lisArgs);

//		错误
//		List<ByteString> lisArgs = new ArrayList<ByteString>();
//		lisArgs.add(ByteString.copyFromUtf8(messageEntity.toJSON().toString()));
//		//异步发送消息
//		stub.invokeChaincode("rbcnotify", "sendMessage", lisArgs);
	}
	
	
	/**
	 * 查询一条消息
	 * @param stub
	 * @param msgId 消息Id
	 */
	public MessageEntity getMessage(ChaincodeStub stub,String msgId){
		if (StringUtil.isEmpty(msgId)){
			return null;
		}
		List<ByteString> lisArgs = new ArrayList<ByteString>();
		lisArgs.add(ByteString.copyFromUtf8(msgId));
		String strReturn = stub.queryChaincode("rbcnotify", "getMessage", lisArgs);
		if (StringUtil.isEmpty(strReturn)){
			return null;
		}
		JSONObject jData = JSONUtil.getJSONObjectFromStr(strReturn);
		if (jData == null){
			return null;
		}
		
		MessageEntity messageEntity = new MessageEntity(jData);
		return messageEntity;
	}
	
	/**
	 * 根据条件查询消息列表
	 * @param securityCert
	 * @param jQuery
	 * @param cpno
	 * @return
	 */
	public PageList<MessageEntity> getMessageList(ChaincodeStub stub,JSONObject jParams,int cpno){
		if (jParams == null){
			jParams = new JSONObject();
		}
		PageList<MessageEntity> lisMessageEntity = new PageList<MessageEntity>();

		jParams.put("cpno", ""+cpno);

		List<ByteString> lisArgs = new ArrayList<ByteString>();
		lisArgs.add(ByteString.copyFromUtf8(jParams.toString()));
		String strReturn = stub.queryChaincode("rbcnotify", "getMessageList", lisArgs);
	
		if (StringUtil.isEmpty(strReturn)){
			return lisMessageEntity;
		}
		JSONObject jData = JSONUtil.getJSONObjectFromStr(strReturn);
		if (jData == null){
			return lisMessageEntity;
		}
		
		lisMessageEntity.fromJSON(jData, MessageEntity.class);
		return lisMessageEntity;
	}
	
	/**
	 * 设置(关闭通知)消息己读
	 * @param stub
	 * @param msgId 消息Id
	 */
	public void readMessage(ChaincodeStub stub,String msgId){
		if (StringUtil.isEmpty(msgId)){
			logger.error("msgId is empty");
			return;
		}
		List<ByteString> lisArgs = new ArrayList<ByteString>();
		lisArgs.add(ByteString.copyFromUtf8(msgId));
		stub.invokeChaincode("rbcnotify", "readMessage", lisArgs);
	}
	
	
	/**
	 * 处理(关闭)消息通知
	 * @param stub
	 * @param msgId 消息Id
	 */
	public void handleMessage(ChaincodeStub stub,String msgId){
		if (StringUtil.isEmpty(msgId)){
			logger.error("msgId is empty");
			return;
		}
		List<ByteString> lisArgs = new ArrayList<ByteString>();
		lisArgs.add(ByteString.copyFromUtf8(msgId));
		stub.invokeChaincode("rbcnotify", "handleMessage", lisArgs);
	}
	

	
	/**
	 * 清空设置某一主键数据的所有消息
	 * @param stub
	 * @param bizId 消息业务Id
	 */
	public void clearMessage(ChaincodeStub stub,String bizId){
		if (StringUtil.isEmpty(bizId)){
			logger.error("bizId is empty");
			return;
		}
		List<ByteString> lisArgs = new ArrayList<ByteString>();
		lisArgs.add(ByteString.copyFromUtf8(bizId));
		stub.invokeChaincode("rbcnotify", "clearMessage", lisArgs);
	}
	
	/**
	 * 字节转byte数组
	 * @param ecPublicKey
	 * @return
	 */
	public static String publicKeyToByte(ECPublicKey ecPublicKey){
		String publicX = "0000"+Hex.encodeHexString(ecPublicKey.getW().getAffineX().toByteArray());
		String publicY = "0000"+Hex.encodeHexString(ecPublicKey.getW().getAffineY().toByteArray());
		if (publicX.length()>64){
			publicX = publicX.substring(publicX.length()-64);
		}
		if (publicY.length()>64){
			publicY = publicY.substring(publicY.length()-64);
		}
		return "04"+publicX+publicY;
	}
	
	//主体公钥的缓存
	private static Map<String,ECPublicKey> Map_MainPublckKey = new HashMap<String,ECPublicKey>();

	/**
	 * 获取主体公钥
	 * @param securityCert
	 * @param modelName
	 * @param mainId
	 * @return
	 */
	public static ECPublicKey getMainPublicKey(ChaincodeStub stub,String modelName,String mainId)
	{
		if (StringUtil.isEmpty(mainId)){
			return null;
		}
		CryptoSuite cryptoSuite = CryptoSuite.Factory.getCryptoSuite();

		String key = "__MAINPUBKEY_"+modelName+"_"+mainId;
		if (Map_MainPublckKey.get(key) != null ){
			return Map_MainPublckKey.get(key);
		}
		
		byte[]  pubKeyBuf = null;
		
		List<ByteString> lisArgs = new ArrayList<ByteString>();
		lisArgs.add(ByteString.copyFromUtf8("getMainPublicKey"));
		lisArgs.add(ByteString.copyFromUtf8(modelName));
		lisArgs.add(ByteString.copyFromUtf8(mainId));
		//从区块链上获取主体公钥
		String strReturn = stub.queryChaincode("rbcmodel", "query", lisArgs);
		if (StringUtil.isEmpty(strReturn)){
			return null;
		}
		ECPublicKey ecPublicKey = null;
		try{
			String pubKeyStr = strReturn;
			pubKeyStr = pubKeyStr.replaceAll("\"", "");
			if (StringUtil.isEmpty(pubKeyStr)){
				return null;
			}
			pubKeyBuf = Hex.decodeHex(pubKeyStr.toCharArray());
			ecPublicKey = cryptoSuite.bytesToPublicKey(pubKeyBuf);
			Map_MainPublckKey.put(key, ecPublicKey);
		}catch(Exception e)
		{
			e.printStackTrace();
		}
		
		return ecPublicKey;
	}
	
	private static KeyPair gKeyPair = null;
	
	/**
	 * 进程级，密码公钥
	 * @return
	 */
	public static String getPubG(){
		String pubG = "";
		try{
			if (gKeyPair == null){
				CryptoSuite cryptoSuite = CryptoSuite.Factory.getCryptoSuite();
				gKeyPair = cryptoSuite.keyGen();
			}
			pubG =  publicKeyToByte((ECPublicKey)gKeyPair.getPublic());
		}catch(Exception e){
			
		}
		return pubG;
	}
	
	/**
	 * 从orderer发送通用http请求
	 * @param stub
	 * @param url
	 * @param jParams
	 * @return
	 */
	public static String getFromHttp(ChaincodeStub stub,String url,JSONObject jParams)
	{
		if (jParams == null){
			jParams = new JSONObject();
		}
		List<ByteString> lisArgs = new ArrayList<ByteString>();
		lisArgs.add(ByteString.copyFromUtf8("getFromHttp"));
		lisArgs.add(ByteString.copyFromUtf8(url));
		lisArgs.add(ByteString.copyFromUtf8(jParams.toString()));
		//从区块链上获取主体公钥
		String strReturn = stub.queryChaincode("rbcmodel", "query", lisArgs);
		if (StringUtil.isEmpty(strReturn)){
			return null;
		}

		return strReturn;
	}
	
	/**
	 * 从orderer获取clob型字段的值
	 * @param stub
	 * @param key
	 * @param jParams
	 * @return
	 */
	public static String getAttach(ChaincodeStub stub,String key)
	{
		if (StringUtil.isEmpty(key)){
			return "";
		}
		List<ByteString> lisArgs = new ArrayList<ByteString>();
		lisArgs.add(ByteString.copyFromUtf8("getAttach"));
		lisArgs.add(ByteString.copyFromUtf8(key));
		//从区块链上获取主体公钥
		String strReturn = stub.queryChaincode( "rbcmodel", "query", lisArgs);
		if (StringUtil.isEmpty(strReturn)){
			return null;
		}

		return strReturn;
	}
	
	/**
	 * 进程级，对密码进行解密
	 * @param bGram
	 * @param mainPub
	 * @return
	 */
	public static byte[] decCryptogram(byte[] bGram,ECPublicKey mainPub){
		try{
			if (gKeyPair == null){
				CryptoSuite cryptoSuite = CryptoSuite.Factory.getCryptoSuite();
				gKeyPair = cryptoSuite.keyGen();
			}

			RBCEncrypto rbcEncrypto= new RBCEncrypto(mainPub,(ECPrivateKey)gKeyPair.getPrivate());
			byte[] theGram = rbcEncrypto.getCryptogram("", "", "", "");

			byte[] decGram = dec(theGram,bGram);
			return decGram;
		}catch(Exception e){
			
		}

		return bGram;
	}
	
	/**
	 * 获取密码
	 * @param securityCert
	 * @param modelName
	 * @param tableName
	 * @param mainId
	 * @param customerNo
	 * @param pubR
	 * @return
	 */
	public static JSONObject getCryptogram(ChaincodeStub stub,String modelName,String tableName,String mainId,String customerNo,String pubR,String colHash){
		JSONObject jData = new JSONObject();
		jData.put("mainId", mainId);
		jData.put("pubR", pubR);
		jData.put("customerNo", customerNo);
		if ("__CHAINCODE".equals(tableName)){
			jData.put("etype", "gram");
			jData.put("chaincodeName", modelName);

		}
		jData.put("colsHash",colHash);
		//从解密中心获取密码
		List<ByteString> lisArgs = new ArrayList<ByteString>();
		lisArgs.add(ByteString.copyFromUtf8("getMainCryptogram"));
		lisArgs.add(ByteString.copyFromUtf8(modelName));
		lisArgs.add(ByteString.copyFromUtf8(tableName));
		lisArgs.add(ByteString.copyFromUtf8(getPubG()));
		lisArgs.add(ByteString.copyFromUtf8(jData.toString()));
		String strReturn = stub.queryChaincode("rbcmodel", "query", lisArgs);
		if (StringUtil.isEmpty(strReturn)){
			logger.error(String.format("getCryptogram queryChaincode is null: %s", lisArgs.toString()));
			return null;
		}
		JSONObject jCryptogram = JSONUtil.getJSONObjectFromStr(strReturn);
		if (jCryptogram == null){
			return null;
		}
		try{
			if (jCryptogram != null && jCryptogram.get("__ENCGRAM")!= null){
				byte[] bCryptogram = Base64.decode(jCryptogram.getString("__ENCGRAM"));
				String thePubR = (String)jCryptogram.get("pubR");

				byte[] decCryptogram = decCryptogram(bCryptogram, CryptoSuite.Factory.getCryptoSuite().bytesToPublicKey(Hex.decodeHex(thePubR.toCharArray())));
				JSONObject jCryptogram1= JSONUtil.getJSONObjectFromStr(new String(decCryptogram));
				
				if (jCryptogram1 != null){
					jCryptogram =  jCryptogram1;
				}
			}
		}catch(Exception e){
			
		}
		
		return jCryptogram;
	}
	
	/**
	 * BaseStatic转ByteString
	 * @return
	 */
	public static ByteString toByteString(Object object)
	{
		
		ByteString byteString = null;
		ByteArrayOutputStream  byteOut = null;
		ObjectOutputStream out = null;
		try 
		{
			byteOut = new ByteArrayOutputStream();
			out = new ObjectOutputStream(byteOut);
			out.writeObject(object);
			byteString = ByteString.copyFrom(byteOut.toByteArray());			
			
		} catch (Exception e) {
			e.printStackTrace();
		}finally
		{
			try
			{
				if (byteOut != null)
				{
					byteOut.flush();
					byteOut.close();
					byteOut = null;
				}
				
				if (out != null)
				{
					out.flush();
					out.close();
					out = null;
				}
			}catch(Exception e)
			{
				
			}
			
		}
		
		return byteString;
		
	}
	

	
	/**
	 * 对内存的克隆处理
	 * @param object
	 * @return
	 */
	public static Object toObject(ByteString byteString)
	{
		
		Object objReturn = null;
		ObjectOutputStream out = null;
		try 
		{
			if (byteString == null)
			{
				return null;
			}
			
			byte[] b= byteString.toByteArray();
			if (byteString == null || b.length<1)
			{
				return null;
			}
			
			ByteArrayInputStream byteIn = new ByteArrayInputStream(b);
			ObjectInputStream in =new ObjectInputStream(byteIn);
			objReturn =in.readObject();
			
			byteIn.close();
			byteIn = null;
			in.close();
			in = null;
			
		} catch (Exception e) {
			e.printStackTrace();
		}finally
		{
			try
			{
				if (out != null)
				{
					out.flush();
					out.close();
					out = null;
				}
			}catch(Exception e)
			{
				
			}
			
		}
		
		return objReturn;
	}
	
	/**
	 * RBC通用的异常处理
	 * @param msg
	 */
	public static void exception(String msg)
	{
		throw new RBCRuntimeException(msg);
	}
	

	/**
	 * hash混合
	 * @param in1
	 * @param in2
	 * @return
	 */
	public static byte[] hash(byte[] in1,byte[] in2 ){
		byte[] bReturn = null;

		try {
			MessageDigest messageDigest = MessageDigest.getInstance("SHA-256");
			messageDigest.update(in1, 0, in1.length);			
			messageDigest.update(in2, 0, in2.length);
			bReturn = messageDigest.digest();
		} catch (Exception  e) {
			e.printStackTrace();
		}
		
		if (bReturn == null || bReturn.length<32) {
			exception("get p2 hash exception");
		}
		return bReturn;
	}
	
	/**
	 * 判断Base64字符串是否加密
	 * @param msg
	 * @return
	 */
	public static boolean isCrypto(String msg)
	{
		boolean bReturn = false;
		if (msg != null  && msg.startsWith("/ty6A"))
		{
			bReturn = true;
		}
		return bReturn;
	}
	
	/**
	 * 判断是否是加密串
	 * @param msg
	 * @return
	 */
	public static boolean isCrypto(byte[] msg)
	{
		boolean bReturn = false;
		if (msg != null && msg.length >3 
				&& msg[0] == (byte)0xFE&& msg[1] == (byte)0xDC&& msg[2] == (byte)0xBA&& msg[3] == (byte)0x00)
		{
			bReturn = true;
		}
		return bReturn;
	}

	/**
	 * 加密
	 * @param msg
	 * @return
	 */
	public static byte[] enc(byte[] cryptogram,byte[] msg){
		if (msg == null || msg.length<1 || isCrypto(msg)){
			return msg;
		}
		byte[] bReturn = new byte[msg.length+4];
		bReturn[0] = (byte)0xFE;
		bReturn[1] = (byte)0xDC;
		bReturn[2] = (byte)0xBA;
		bReturn[3] = (byte)0x00;

		int c = 0;
		byte[] p2 =hash(cryptogram,(""+c).getBytes());

		int pos = 0;
		for (int i=0;i<msg.length;i++){
			bReturn[i+4] = msg[i];			
			bReturn[i+4] ^= p2[pos];
			pos ++;
			if (pos >= 32 ){
				//大数据流加密提高性能
				//if (c <=6){
					c ++;
					p2 = hash(cryptogram,(""+c).getBytes());
				//}
				pos = 0;
			}
		}
		
		return bReturn;
	}
	
	/**
	 * 解密
	 * @param msg
	 * @return
	 */
	public static byte[] dec(byte[] cryptogram,byte[] msg){
		if (msg == null || msg.length<5 || !isCrypto(msg)){
			return msg;
		}
		
		int c = 0;

		byte[] p2 =hash(cryptogram,(""+c).getBytes());
		byte[] bReturn = new byte[msg.length-4];

		int pos = 0;
		for (int i=4;i<msg.length;i++){
			bReturn[i-4] = msg[i];			
			bReturn[i-4] ^= p2[pos];	
			pos ++;
			if (pos >= 32 ){
				//大数据流加密提高性能
				//if (c <=6){
					c ++;
					p2 = hash(cryptogram,(""+c).getBytes());
				//}
				pos = 0;
			}
		}
		
		return bReturn;
	}
	
}
