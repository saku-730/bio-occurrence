"use client";

import { createContext, useContext, useState, useEffect, ReactNode } from "react";
import { useRouter } from "next/navigation";

// ユーザー情報の型 (Goのモデルと合わせる)
type User = {
  id: string;
  username: string;
  email: string;
};

type AuthContextType = {
  user: User | null;
  login: (email: string, pass: string) => Promise<void>;
  register: (name: string, email: string, pass: string) => Promise<void>;
  logout: () => void;
  isLoading: boolean;
};

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const router = useRouter();

  // 起動時にlocalStorageからユーザー情報を復元する
  useEffect(() => {
    const savedUser = localStorage.getItem("bio_user");
    if (savedUser) {
      try {
        setUser(JSON.parse(savedUser));
      } catch (e) {
        console.error("ユーザー情報の復元に失敗", e);
        localStorage.removeItem("bio_user");
      }
    }
    setIsLoading(false);
  }, []);

  // ログイン処理
  const login = async (email: string, pass: string) => {
    const res = await fetch("http://localhost:8080/api/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password: pass }),
    });

    if (!res.ok) {
      const err = await res.json();
      throw new Error(err.error || "ログイン失敗");
    }

    const data = await res.json();
    const userData = data.user;

    // 状態を更新して保存
    setUser(userData);
    localStorage.setItem("bio_user", JSON.stringify(userData));
    router.push("/"); // トップページへ
  };

  // 登録処理
  const register = async (name: string, email: string, pass: string) => {
    const res = await fetch("http://localhost:8080/api/auth/register", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username: name, email, password: pass }),
    });

    if (!res.ok) {
      const err = await res.json();
      throw new Error(err.error || "登録失敗");
    }

    // 登録後はそのままログインさせるか、ログイン画面に飛ばす
    // ここではログイン画面に飛ばす
    router.push("/login");
  };

  // ログアウト処理
  const logout = () => {
    setUser(null);
    localStorage.removeItem("bio_user");
    router.push("/login");
  };

  return (
    <AuthContext.Provider value={{ user, login, register, logout, isLoading }}>
      {children}
    </AuthContext.Provider>
  );
}

// 使いやすくするためのフック
export const useAuth = () => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
};
