package com.rongzer.chaincode.base;

import java.util.ArrayList;
import java.util.Date;
import java.util.Iterator;
import java.util.List;

import net.sf.json.JSONArray;
import net.sf.json.JSONObject;

import org.apache.commons.codec.binary.Base64;
import org.apache.log4j.MDC;
import com.rongzer.blockchain.protos.common.Rbc.RListHeight;
import com.rongzer.blockchain.shim.ChaincodeBase;
import com.rongzer.blockchain.shim.ChaincodeStub;
import com.rongzer.blockchain.shim.ledger.KeyModification;
import com.rongzer.blockchain.shim.ledger.QueryResultsIterator;

import com.rongzer.chaincode.entity.CustomerEntity;
import com.rongzer.chaincode.entity.HistoryEntity;
import com.rongzer.chaincode.entity.PageList;
import com.rongzer.chaincode.entity.RList;
import com.rongzer.chaincode.entity.TableDataEntity;
import com.rongzer.chaincode.utils.ChainCodeUtils;
import com.rongzer.chaincode.utils.JSONUtil;
import com.rongzer.chaincode.utils.ModelUtils;
import com.rongzer.chaincode.utils.StringUtil;

/**
 * RBC智能合约基类，实现run、query、getChaincodeID方法
 * @author Administrator
 *
 */
public abstract class RBCChaincodeBase extends ChaincodeBase {
	
	public abstract Response run(ChaincodeStub stub, String function, String[] args);
	public abstract Response query(ChaincodeStub stub, String function, String[] args);
    public abstract String getChaincodeID();

	@Override
	public Response init(ChaincodeStub stub) {
		final List<String> argList = stub.getStringArgs();
		final String function = argList.get(0);
		final String[] args = argList.stream().skip(1).toArray(String[]::new);

		return run(stub,function,args);
	}

	@Override
	public Response invoke(ChaincodeStub stub) {
		final List<String> argList = stub.getStringArgs();
		final String method = argList.get(0);
		final String function = argList.get(1);
		final String[] args = argList.stream().skip(2).toArray(String[]::new);
		Response response = null;
		
		int defaultExecute = 0;
		//rbc model execute
		if (args.length >2 && "__setTableData".equals(function))
		{
			try{
				//多条
				if (args[2].startsWith("["))
				{
					JSONArray jArray = JSONUtil.getJSONArrayFromStr(args[2]);
					if (jArray != null) {
						boolean bReturn = false;
						//处理预读并发，提升批量插入性能
						List<String> lisIdKey = new ArrayList<String>();
						for (int i = 0; i < jArray.size(); i++) {
							JSONObject jObject = jArray.getJSONObject(i);
							String idKey = ModelUtils.getTableDataKey(stub,  args[0], args[1],  jObject);
							if (StringUtil.isNotEmpty(idKey)){
								lisIdKey.add(idKey);
							}
						}
						stub.getStringStates(lisIdKey);
						//预读处理结束
						
						for (int i=0;i<jArray.size();i++){
							bReturn = ModelUtils.setTableData(stub,  args[0], args[1], jArray.getJSONObject(i));
							defaultExecute ++;
							if (!bReturn) {
								break;
							}
						}

						if (!bReturn) 
						{
							MDC.clear();
							return newErrorResponse(String.format("set model %s table %s data reutun false",args[0], args[1]));
						}
						
					}
				}else{
					JSONObject jData = JSONUtil.getJSONObjectFromStr(args[2]);
					if (jData != null) {
						boolean bReturn = ModelUtils.setTableData(stub,  args[0], args[1], jData);
						defaultExecute ++;

						if (!bReturn) 
						{
							MDC.clear();
							return newErrorResponse(String.format("set model %s table %s data reutun false",args[0], args[1]));
						}
					}
				}
				
			}catch(Exception e)
			{
				e.printStackTrace();
			}
		}

		if (args.length >2 && "__setTableDataSupplement".equals(function))
		{
			try{
				//多条
				if (args[2].startsWith("["))
				{
  					JSONArray jArray = JSONUtil.getJSONArrayFromStr(args[2]);
					if (jArray != null) {
						boolean bReturn = false;
						//处理预读并发，提升批量插入性能
						/*List<String> lisIdKey = new ArrayList<String>();
						for (int i = 0; i < jArray.size(); i++) {
							JSONObject jObject = jArray.getJSONObject(i);
							String idKey = ModelUtils.getTableDataKey(stub,  args[0], args[1],  jObject);
							if (StringUtil.isNotEmpty(idKey)){
								lisIdKey.add(idKey);
							}
						}
						stub.getStringStates(lisIdKey);*/
						//预读处理结束

						for (int i=0;i<jArray.size();i++){
							bReturn = ModelUtils.setTableDataSupplement(stub,jArray.getJSONObject(i));
							defaultExecute ++;
							if (!bReturn) {
								break;
							}
						}

						if (!bReturn)
						{
							MDC.clear();
							return newErrorResponse(String.format("set model %s table %s data reutun false",args[0], args[1]));
						}

						ModelUtils.setTableSupplementHeight(stub,args[0],args[1],args[3]);
					}



				}else{
					JSONObject jData = JSONUtil.getJSONObjectFromStr(args[2]);
					if (jData != null) {
						boolean bReturn = ModelUtils.setTableDataSupplement(stub, jData);
						defaultExecute ++;

						if (!bReturn)
						{
							MDC.clear();
							return newErrorResponse(String.format("set model %s table %s data reutun false",args[0], args[1]));
						}
						ModelUtils.setTableSupplementHeight(stub,args[0],args[1],args[3]);
					}
				}

			}catch(Exception e)
			{
				e.printStackTrace();
			}
		}

		if (args.length ==2 && "__getTableSupplementHeight".equals(function)) {
			try{
				//执行动作，验证操作者身份是否有效
				String height = ModelUtils.getTableSupplementHeight(stub, args[0], args[1]);
				if (null != height){
					response = newSuccessResponse(height.getBytes());
				}else{
					response = newSuccessResponse();
				}
				MDC.clear();

				return response;
			}catch(Exception e)
			{
				response = newErrorResponse(e.getMessage());
				MDC.clear();

				return response;
			}

		}

		if (args.length  == 1 && "__getHistoryForKey".equals(function)) {
			try{
				JSONArray keyModifications = ModelUtils.getHistoryForKey(stub, args[0]);
				if (null != keyModifications){
					response = newSuccessResponse(keyModifications.toString().getBytes("UTF-8"));
				}else{
					response = newSuccessResponse();
				}
				MDC.clear();

				return response;
			}catch(Exception e)
			{
				response = newErrorResponse(e.getMessage());
				MDC.clear();

				return response;
			}
		}

		if (args.length >2 && "__delTableData".equals(function))
		{
			try{
				boolean bReturn = ModelUtils.delTableData(stub,args[0], args[1], args[2]);
				defaultExecute ++;

				if (!bReturn) 
				{
					MDC.clear();

					return newErrorResponse(String.format("delete model %s table %s data id:%s reutun false",args[0], args[1], args[2]));
				}
			}catch(Exception e)
			{
				e.printStackTrace();
			}
		}
		
		if (args.length >2 && "__getTableData".equals(function))
		{
			try{
				//执行动作，验证操作者身份是否有效
				TableDataEntity tableDataEntity = ModelUtils.getTableData(stub, args[0], args[1], args[2]);
				if (tableDataEntity != null){
					tableDataEntity.dec(stub);
					response = newSuccessResponse(tableDataEntity.getBytes());				
				}else{
					response = newSuccessResponse();	
				}			
				MDC.clear();

				return response;
			}catch(Exception e)
			{
				response = newErrorResponse(e.getMessage());
				MDC.clear();

				return response;
			}
		}
		
		
		if (args.length ==3 && "__lisTableData".equals(function))
		{
			try{
				PageList<TableDataEntity> lisTableData = ModelUtils.lisTableData(stub, args[0], args[1], StringUtil.toInt(args[2],0));
				lisTableData.dec(stub);
				
				response = newSuccessResponse(lisTableData.getBytes());
				MDC.clear();

				return response;
			}catch(Exception e)
			{
				response = newErrorResponse(e.getMessage());
				return response;
			}
		}
		
		if (args.length ==4 && "__lisTableData".equals(function))
		{
			try{
				PageList<TableDataEntity> lisTableData = ModelUtils.lisTableData(stub, args[0], args[1],args[2], StringUtil.toInt(args[3],0),0);
				lisTableData.dec(stub);
				response = newSuccessResponse(lisTableData.getBytes());
				MDC.clear();

				return response;
			}catch(Exception e)
			{
				response = newErrorResponse(e.getMessage());
				MDC.clear();

				return response;
			}
		}
		
		if (args.length ==1 &&"__queryTableDataByHeight".equals(function))//根据高度查询表数据信息
		{
			
			//获取参数，转移成map[string]string
			String argString = (String) args[0];
			JSONObject jParam = JSONUtil.getJSONObjectFromStr(argString);
			String modelName = jParam.get("modelName")==null?"":(String)jParam.get("modelName");
			String tableName = jParam.get("tableName")==null?"":(String)jParam.get("tableName");
			String needDec = jParam.get("dec")==null?"":(String)jParam.get("dec");
			String needDetail = jParam.get("needDetail")==null?"":(String)jParam.get("needDetail");

			int height = StringUtil.toInt((String)jParam.get("height"), 0);
			String rListName = "__IDX_" + ModelUtils.buildKey(modelName, tableName,"");
			
			JSONObject result = new JSONObject();
			JSONArray jsonArray = new JSONArray();

			int maxCount =  StringUtil.toInt((String)jParam.get("maxCount"), 100);//最大处理约100条+1块
			try {
				RList rList = RList.getInstance(stub, rListName);
				int nSize = rList.size();
				result.put("rnum", nSize+"");
				PageList<TableDataEntity> lisTableData = new PageList<TableDataEntity>();
				while (height>=0 && maxCount>0)
				{
					RListHeight rListHeight = rList.getIdsByHeight(height);
					result.put("nextHeight", (int)rListHeight.getNextHeight()+"");
					height = (int)rListHeight.getNextHeight();
					maxCount-=rListHeight.getIdsCount();

					for (int i=0;i<rListHeight.getIdsCount();i++){
						JSONObject jObjectR = new JSONObject();
						String va = rListHeight.getIds(i);
						String r = new String(stub.getState(va), "UTF-8");
						if(StringUtil.isEmpty(r)){
							continue;
						}
						jObjectR = JSONUtil.getJSONObjectFromStr(r);
						if(jObjectR == null){
							continue;
						}
						TableDataEntity tableDataEntity = new TableDataEntity(jObjectR);
						if (!"false".equals(needDetail)) {
							tableDataEntity.processAttach(stub);
						}

						lisTableData.add(tableDataEntity);
						//tableDataEntity.dec(stub);
						//jsonArray.add(tableDataEntity.toJSON());
					}
				}
				if (!"false".equals(needDec)){
					lisTableData.dec(stub);					
				}
				
				for (TableDataEntity tableDataEntity:lisTableData){
					jsonArray.add(tableDataEntity.toJSON());
				}
				result.put("psize", jsonArray.size()+"");
				result.put("msg", jsonArray);
				response = newSuccessResponse(result.toString().getBytes());
			} catch (Exception e) {
				e.printStackTrace();
				response = newErrorResponse(e.toString());
			}
			MDC.clear();

			return response;
		}
		
		
		if ("query".equals(method) || "query".equals(function) || function.toLowerCase().startsWith("query")|| function.toLowerCase().startsWith("get")|| function.toLowerCase().startsWith("lis"))
		{
			if ("queryRBCCName".equals(function))//默认查询合约名称方法
			{
				response = newSuccessResponse(stub.getSecurityContext().getChainCodeName().getBytes());
			}else if ("queryStateHistory".equals(function))//默认查询状态值历史方法
			{
				if (args.length != 1)
				{
					newErrorResponse("query state history need 1 args");
				}
				
				QueryResultsIterator<KeyModification>  iter = stub.getHistoryForKey(args[0]);
				
				Iterator<KeyModification> iterator = iter.iterator();
				PageList<HistoryEntity> pageList = new PageList<HistoryEntity>();
				
				while (iterator.hasNext() ) {
					KeyModification keyModification = iterator.next();
					HistoryEntity historyEntity = new HistoryEntity();
					historyEntity.setTxId(keyModification.getTxId());					
					historyEntity.setTxTime(StringUtil.dateTimeToStr(new Date(keyModification.getTimestamp().toEpochMilli())));
					historyEntity.setValue(Base64.encodeBase64String(keyModification.getValue()));
					pageList.add(historyEntity);
					if (pageList.size()>100)
					{
						pageList.remove(0);
					}
				}
				
				response = newSuccessResponse(pageList.getBytes());
				
			}else if ("queryStateRList".equals(function))//默认查询某一个RList列表
			{
				if (args.length != 2)
				{
					newErrorResponse("query state history need 2 args");
				}
				

				//获取参数，转移成map[string]string
				String argString = (String) args[1];
				JSONObject jParam = JSONUtil.getJSONObjectFromStr(argString);
				
				int cpno = StringUtil.toInt((String)jParam.get("cpno"), 0);
				
				RList rList = RList.getInstance(stub,args[0]);
				int start = cpno*100;
				int end = (cpno+1) * 100 ;
				if (end >rList.size())
				{
					end = rList.size();
				}
				
				PageList<CustomerEntity> lisStateRList = new PageList<CustomerEntity>();
				lisStateRList.setRnum(rList.size());
				lisStateRList.setCpno(cpno);
				int nSize = rList.size();
				
				for (int i=start;i<end;i++)
				{
					String customerId = rList.get(nSize - i - 1);
					String customerNo = ""+rList.getHeightById(customerId);
					CustomerEntity cusomerEntity = new CustomerEntity();
					cusomerEntity.setCustomerId(customerId);
					cusomerEntity.setCustomerNo(customerNo);
					lisStateRList.add(cusomerEntity);
				}
				
				response = newSuccessResponse(lisStateRList.getBytes());
				
			}else 
			{
				try{
					response = query(stub,function,args);
				}catch(Exception e)
				{
					response = newErrorResponse(e.toString());
				}
			}
			
		}else
		{
			try{
				//执行动作，验证操作者身份是否有效
				CustomerEntity customerEntity = ChainCodeUtils.queryExecCustomer(stub);
				if (customerEntity == null)
				{
					MDC.clear();

					return newErrorResponse("tx customer is not exist ");
				}
				
				if (!"1".equals(customerEntity.getCustomerStatus()))
				{
					MDC.clear();

					return newErrorResponse("tx customer status "+customerEntity.getCustomerStatus()+" is invalid");
				}
				
				response = run(stub,function,args);
				/*
				//rongzer 清除MDC线程对象池
				Hashtable hmdc = MDC.getContext();
				if (hmdc != null)
				{
					Iterator keys = hmdc.keySet().iterator();
					while (keys.hasNext()) {
						String key = (String) keys.next();
						Object value = hmdc.get(key);
						if(value instanceof RList)
						{
							try
							{
								RList rList = (RList)value;
								rList.saveState();
							}catch(Exception e1)
							{
								e1.printStackTrace();
							}
						}
					}
					
				}
				*/
			}catch(Exception e)
			{
				response = newErrorResponse(e.toString());
			}

		}
		MDC.clear();

		if ((response == null || response.getStatus().getCode() != 200) && defaultExecute >0) {
			response = newSuccessResponse("default execute success".getBytes());
		}
		
		if (response.getStatus().getCode() != 200){
			logger.error(String.format("[%-8s] ChainCode=%s execute error: %s", stub.getTxId(),stub.getSecurityContext().getChainCodeName(), response.getMessage()));
		}
		return response;
	}

}
