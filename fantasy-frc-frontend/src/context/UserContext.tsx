import { createContext, useContext, useState, ReactNode } from 'react';

type User = {
  name: string;
  isLoggedIn: boolean;
};

type UserContextType = {
  user: User;
  logout: () => void;
  loginAs: (name: string) => void;
};

const defaultUser = {
  name: '',
  isLoggedIn: false,
};

const UserContext = createContext<UserContextType | null>(null);

export const useUser = () => {
  const ctx = useContext(UserContext);
  if (!ctx) throw new Error('UserContext must be used within UserProvider');
  return ctx;
};

export const UserProvider = ({ children }: { children: ReactNode }) => {
  const [user, setUser] = useState<User>(defaultUser);

  const loginAs = (name: string) => setUser({ name, isLoggedIn: true });
  const logout = () => setUser(defaultUser);

  return (
    <UserContext.Provider value={{ user, logout, loginAs }}>
      {children}
    </UserContext.Provider>
  );
};

