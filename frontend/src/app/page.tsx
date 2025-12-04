import Link from "next/link";
import { Database, PlusCircle, List, Search } from "lucide-react";

export default function Home() {
  return (
    <main className="min-h-screen bg-gray-50 flex flex-col items-center justify-center p-4">
      <div className="max-w-4xl w-full text-center space-y-12">
        
        {/* ヒーローセクション */}
        <div className="space-y-4">
          <div className="inline-flex p-4 bg-blue-100 rounded-full text-blue-600 mb-4">
            <Database className="w-16 h-16" />
          </div>
          <h1 className="text-5xl font-black text-gray-900 tracking-tight">
            Bio Occurrence DB
          </h1>
          <p className="text-xl text-gray-500 max-w-2xl mx-auto">
            生物のオカレンスデータを集約・管理・検索するための<br className="hidden sm:inline" />
            ソフト・プラットフォーム。
          </p>
        </div>

        {/* メニューカード */}
        <div className="grid md:grid-cols-2 gap-6 max-w-2xl mx-auto">
          {/* 登録ボタン */}
          <Link 
            href="/create" 
            className="group block bg-white p-8 rounded-2xl shadow-sm border border-gray-200 hover:border-blue-500 hover:shadow-md transition-all text-left"
          >
            <div className="flex items-center gap-4 mb-4">
              <div className="p-3 bg-blue-50 text-blue-600 rounded-lg group-hover:bg-blue-600 group-hover:text-white transition-colors">
                <PlusCircle className="w-8 h-8" />
              </div>
              <h2 className="text-2xl font-bold text-gray-800">データを登録</h2>
            </div>
            <p className="text-gray-500">
              新しいオカレンスデータを登録する
            </p>
          </Link>

          {/* 一覧ボタン */}
          <Link 
            href="/occurrences" 
            className="group block bg-white p-8 rounded-2xl shadow-sm border border-gray-200 hover:border-purple-500 hover:shadow-md transition-all text-left"
          >
            <div className="flex items-center gap-4 mb-4">
              <div className="p-3 bg-purple-50 text-purple-600 rounded-lg group-hover:bg-purple-600 group-hover:text-white transition-colors">
                <List className="w-8 h-8" />
              </div>
              <h2 className="text-2xl font-bold text-gray-800">データを探す</h2>
            </div>
            <p className="text-gray-500">
              登録されたデータを検索・閲覧する
            </p>
          </Link>
        </div>

      </div>
    </main>
  );
}
