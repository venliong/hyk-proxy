/**
 * 
 */
package com.hyk.proxy.gae.client.netty;

import java.io.FileInputStream;
import java.net.InetAddress;
import java.net.InetSocketAddress;
import java.net.UnknownHostException;
import java.security.KeyStore;
import java.util.HashMap;
import java.util.LinkedList;
import java.util.List;
import java.util.Map;
import java.util.concurrent.Executor;
import java.util.concurrent.Executors;

import javax.net.ssl.KeyManager;
import javax.net.ssl.KeyManagerFactory;
import javax.net.ssl.SSLContext;
import javax.net.ssl.TrustManager;
import javax.net.ssl.TrustManagerFactory;

import org.jboss.netty.bootstrap.ServerBootstrap;
import org.jboss.netty.channel.socket.nio.NioServerSocketChannelFactory;
import org.jivesoftware.smack.XMPPException;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.hyk.proxy.gae.client.config.Config;
import com.hyk.proxy.gae.client.config.XmppAccount;
import com.hyk.proxy.gae.client.http.HttpClientRpcChannel;
import com.hyk.proxy.gae.client.xmpp.XmppRpcChannel;
import com.hyk.proxy.gae.common.HttpServerAddress;
import com.hyk.proxy.gae.common.XmppAddress;
import com.hyk.proxy.gae.common.service.FetchService;
import com.hyk.rpc.core.RPC;
import com.hyk.rpc.core.service.NameService;

/**
 * @author Administrator
 * 
 */
public class HttpServer
{
	protected static Logger	logger	= LoggerFactory.getLogger(HttpServer.class);

	public static SSLContext initSSLContext() throws Exception
	{
		String password = "hykproxy";
		SSLContext sslContext = SSLContext.getInstance("TLS");
		KeyManagerFactory kmf = KeyManagerFactory.getInstance(KeyManagerFactory.getDefaultAlgorithm());
		KeyStore ks = KeyStore.getInstance(KeyStore.getDefaultType());
		ks.load(HttpServer.class.getResourceAsStream("/hykproxykeystore"), password.toCharArray());
		kmf.init(ks, password.toCharArray());
		KeyManager[] km = kmf.getKeyManagers();
		TrustManagerFactory tmf = TrustManagerFactory.getInstance(TrustManagerFactory.getDefaultAlgorithm());
		tmf.init(ks);
		TrustManager[] tm = tmf.getTrustManagers();
		sslContext.init(km, tm, null);
		return sslContext;
	}

	private static RPC createXmppRpc(XmppAccount account) throws XMPPException
	{
		XmppRpcChannel xmppRpcchannle = new XmppRpcChannel(Executors.newFixedThreadPool(3), account.getName(), account.getPasswd());
		return new RPC(xmppRpcchannle);
	}

	private static RPC createHttpRpc(String appid)
	{
		HttpServerAddress remoteAddress = new HttpServerAddress(appid + ".appspot.com", "/fetchproxy");
		HttpClientRpcChannel httpCleintRpcchannle = new HttpClientRpcChannel(Executors.newFixedThreadPool(10), remoteAddress, 1048576);
		return new RPC(httpCleintRpcchannle);
	}

	private static FetchService initXmppFetchService(String appid, RPC rpc) throws XMPPException
	{
		NameService serv = rpc.getRemoteNaming(new XmppAddress(appid + "@appspot.com"));
		FetchService fetchService = (FetchService)serv.lookup("fetch");
		return fetchService;
	}

	private static FetchService initHttpFetchService(String appid, RPC rpc) throws XMPPException
	{
		HttpServerAddress remoteAddress = new HttpServerAddress(appid + ".appspot.com", "/fetchproxy");
		NameService serv = rpc.getRemoteNaming(remoteAddress);
		return (FetchService)serv.lookup("fetch");
	}

	public static void main(String[] args) throws UnknownHostException
	{
		List<FetchService> fetchServices = new LinkedList<FetchService>();
		SSLContext sslContext = null;
		Config config = new Config();
		try
		{
			sslContext = initSSLContext();
			config.loadConfig();
			List<String> appids = config.getAppids();
			if(config.isXmppEnable())
			{
				List<XmppAccount> xmppAccounts = config.getAccounts();
				for(XmppAccount account : xmppAccounts)
				{
					RPC rpc = createXmppRpc(account);
					for(String appid : appids)
					{
						try
						{
							fetchServices.add(initXmppFetchService(appid, rpc));
						}
						catch(Exception e)
						{
							logger.error("Failed to retireve xmpp remote service, please check your configuration.", e);
						}

					}
				}
			}

			if(config.isHttpEnable())
			{
				for(String appid : appids)
				{					
					try
					{
						RPC rpc = createHttpRpc(appid);
						fetchServices.add(initHttpFetchService(appid, rpc));
					}
					catch(Exception e)
					{
						logger.error("Failed to retireve http remote service, please check your configuration.", e);
					}
					
				}
			}
		}
		catch(Exception e)
		{
			logger.error("Failed to retireve remote service, please check your configuration.", e);
		}
		if(fetchServices.isEmpty())
		{
			logger.error("No fetch service found, please check configuration again.");
			System.exit(-1);
		}
		if(logger.isInfoEnabled())
		{
			logger.info("Found " + fetchServices.size() + " remote fetch service for this proxy.");
		}

		Executor bossExecutor = Executors.newFixedThreadPool(5);
		Executor workerExecutor = Executors.newFixedThreadPool(15);
		ServerBootstrap bootstrap = new ServerBootstrap(new NioServerSocketChannelFactory(bossExecutor, workerExecutor));

		// Set up the event pipeline factory.
		bootstrap.setPipelineFactory(new HttpServerPipelineFactory(fetchServices, workerExecutor, sslContext));
		Map<String, Object> connectionParams = new HashMap<String, Object>();
		bootstrap.bind(new InetSocketAddress(InetAddress.getByName(config.getLocalServerHost()), config.getLocalServerPort()));
	}

}
