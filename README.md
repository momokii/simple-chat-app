# Simple Real Time Web Chat App

This repository contains the source code for a simple chat application that implements the WebSocket protocol. The application is built using HTML, CSS, jQuery, and is served using the Go Fiber web framework. This project serves as a demonstration of WebSocket usage in a real-time chat app.

Feel free to explore the code, suggest improvements, or use it as inspiration for your own project!

## **New Feature: Credit System Integration**
This project now includes a **credit system** integrated with the SSO authentication. The credit tokens can be used to create and access the Dating App Chat Simulation. The credit system ensures fair usage and allows users to manage their usage effectively.

## **How the Credit System Works**
- Users must authenticate via the SSO system.
- Credit tokens can be used to create and access features of the Dating App Chat Simulation, with a cost of 10 credit tokens per create room.
- Credits can be reset daily with user will give MAX 15 token.
- The system ensures fair usage and prevents excessive resource consumption.

This update ensures that resource-heavy features remain accessible while maintaining an efficient and fair usage model.

---
For further details on implementation, refer to the [go-sso-web repository](https://github.com/momokii/go-sso-web).

## Features
- Simple, responsive design with HTML, CSS, and jQuery.
- Real-time messaging using WebSocket.
- Lightweight server built with the Go Fiber framework.
- Dating App Chat Simulation with LLM (OpenAI)
- **Integrated with Single Sign-On (SSO)** for user authentication.  
  (SSO implementation can be found in [go-sso-web repository](https://github.com/momokii/go-sso-web)).

## Related Projects
- [go-sso-web](https://github.com/momokii/go-sso-web): A repository for the custom Single Sign-On (SSO) implementation integrated into this chat application.

## How to Run

1. **Clone the repository:**
   ```bash
   git clone github.com/momokii/simple-chat-app
   ```
   
2. **Configure WebSocket Protocol:**
   - You can choose between `ws` (WebSocket) or `wss` (WebSocket over SSL).
   - If you choose `wss`, ensure you're using the `https` protocol.
   - If needed, generate a self-signed TLS certificate by running the following:
     ```bash
     sh gencert.sh
     ```

3. **Install Go:**
   - Make sure Go is installed. You can check by running:
     ```bash
     go version
     ```

4. **Run the server:**
   - Start the server using the following command:
     ```bash
     go run main.go
     ```

5. **Optional: Use Air for Hot Reloading:**
   - If you want hot reloading during development, you can use [Air](https://github.com/cosmtrek/air).
   - Start the server with Air by running:
     ```bash
     air
     ```

6. **Access the website:**
   - Open your browser and go to `http://localhost:3000` (or the specified port).
