import { createContext, useContext, useState } from 'react';

const ChatLayoutContext = createContext({
    mobileChatOpen: false,
    setMobileChatOpen: () => {},
});

export function ChatLayoutProvider({ children }) {
    const [mobileChatOpen, setMobileChatOpen] = useState(false);
    return (
        <ChatLayoutContext.Provider value={{ mobileChatOpen, setMobileChatOpen }}>
            {children}
        </ChatLayoutContext.Provider>
    );
}

export function useChatLayout() {
    return useContext(ChatLayoutContext);
}
