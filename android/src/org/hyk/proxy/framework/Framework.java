/**
 * This file is part of the hyk-proxy-framework project.
 * Copyright (c) 2010 Yin QiWen <yinqiwen@gmail.com>
 *
 * Description: Framework.java 
 *
 * @author yinqiwen [ 2010-8-12 | 09:28:05 PM]
 *
 */
package org.hyk.proxy.framework;

import java.util.concurrent.ThreadPoolExecutor;

import org.hyk.proxy.framework.common.Constants;
import org.hyk.proxy.framework.common.Misc;
import org.hyk.proxy.framework.config.Config;
import org.hyk.proxy.framework.event.HttpProxyEventServiceFactory;
import org.hyk.proxy.framework.httpserver.HttpLocalProxyServer;
import org.hyk.proxy.framework.management.ManageResource;
import org.hyk.proxy.framework.management.UDPManagementServer;
import org.hyk.proxy.framework.prefs.Preferences;
import org.hyk.proxy.framework.trace.Trace;
import org.jboss.netty.handler.execution.OrderedMemoryAwareThreadPoolExecutor;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 *
 */
public class Framework implements ManageResource
{
	protected Logger logger = LoggerFactory.getLogger(getClass());

	private HttpLocalProxyServer server;
	private UDPManagementServer commandServer;
	private HttpProxyEventServiceFactory esf = null;
//	private PluginManager pm ;
//	private Updater updater;

	private boolean isStarted = false;
	
	private Trace trace;

	public Framework(Trace trace)
	{
		this.trace = trace;
		Preferences.init();
//		pm = PluginManager.getInstance();
		Config config = Config.loadConfig();
		ThreadPoolExecutor workerExecutor = new OrderedMemoryAwareThreadPoolExecutor(
		        config.getThreadPoolSize(), 0, 0);
		Misc.setGlobalThreadPool(workerExecutor);
		Misc.setTrace(trace);
//		pm.loadPlugins(trace);
//		pm.activatePlugins(trace);
//		updater = new Updater(this);
	}

	public void stop()
	{
		try
		{
			if (null != commandServer)
			{
				commandServer.stop();
				commandServer = null;
			}
			if (null != server)
			{
				server.close();
				server = null;
			}
			if (null != esf)
			{
				esf.destroy();
			}
			isStarted = false;
		}
		catch (Exception e)
		{
			logger.error("Failed to stop framework.", e);
		}

	}
	
	public boolean isStarted()
	{
		return isStarted;
	}

	public boolean start()
	{
		return restart();
	}

	public boolean restart()
	{
		try
		{
			stop();
			Config config = Config.loadConfig();
			esf = HttpProxyEventServiceFactory.Registry
			        .getHttpProxyEventServiceFactory(config
			                .getProxyEventServiceFactory());
			if (esf == null)
			{
				logger.error("No event service factory found with name:"
				        + config.getProxyEventServiceFactory());
				return false;
			}
			esf.init();
			server = new HttpLocalProxyServer(
			        config.getLocalProxyServerAddress(),
			        Misc.getGlobalThreadPool(), esf);
			commandServer = new UDPManagementServer(
			        config.getLocalProxyServerAddress());
			Misc.setManagementServer(commandServer);
			commandServer.addManageResource(this);
//			commandServer.addManageResource(pm);
			Misc.getGlobalThreadPool().execute(commandServer);
			trace.notice("Local HTTP Server Running...\nat "
			        + config.getLocalProxyServerAddress());
			isStarted = true;
			return true;
		}
		catch (Exception e)
		{
			logger.error("Failed to launch local proxy server.", e);
		}
		return false;
	}

	@Override
	public String handleManagementCommand(String cmd)
	{
		if (cmd.equals(Constants.STOP_CMD))
		{
			System.exit(1);
		}
		return null;
	}

	@Override
	public String getName()
	{
		return Constants.FRAMEWORK_NAME;
	}
}