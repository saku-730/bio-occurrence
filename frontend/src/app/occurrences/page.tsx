"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Loader2, Bug, Search, Database } from "lucide-react";

// リスト表示用のデータ型
type ListItem = {
  id: string;
  taxon_label: string;
  remarks: string;
  owner_name?: string;
};

export default function OccurrenceList() {
  const [list, setList] = useState<ListItem[]>([]);
  const [loading, setLoading] = useState(true);
  
  // ★検索用の状態管理
  const [searchQuery, setSearchQuery] = useState("");
  const [debounceTimer, setDebounceTimer] = useState<NodeJS.Timeout | null>(null);

  // データ取得関数（検索ワードあり・なしで分岐）
  const fetchData = async (query: string) => {
    setLoading(true);
    try {
      // クエリがあるなら検索API、なければ一覧API
      const url = query 
        ? `http://localhost:8080/api/search?q=${encodeURIComponent(query)}`
        : "http://localhost:8080/api/occurrences";

      const res = await fetch(url);
      if (!res.ok) throw new Error("取得失敗");
      const data = await res.json();
      
      // 検索結果も一覧も、必要なフィールド(id, taxon_label)は共通なのでそのままセット
      setList(data || []);
    } catch (err) {
      console.error(err);
      setList([]);
    } finally {
      setLoading(false);
    }
  };

  // 初回ロード（全件表示）
  useEffect(() => {
    fetchData("");
  }, []);

  // 検索入力のハンドリング (デバウンス処理)
  const handleSearch = (e: React.ChangeEvent<HTMLInputElement>) => {
    const val = e.target.value;
    setSearchQuery(val);

    // 前のタイマーを消す（連打対策）
    if (debounceTimer) clearTimeout(debounceTimer);

    // 0.3秒待ってから検索実行
    const timer = setTimeout(() => {
      fetchData(val);
    }, 300);
    setDebounceTimer(timer);
  };

  // ヘルパー: URIからUUID抽出
  const getUUID = (uri: string) => uri.split("/").pop() || "";

  return (
    <main className="min-h-screen bg-gray-50 py-10 px-4">
      <div className="max-w-4xl mx-auto">
        
        {/* ヘッダー + 検索バーエリア */}
        <div className="flex flex-col md:flex-row justify-between items-center mb-8 gap-4">
          <h1 className="text-3xl font-bold text-gray-800 flex items-center gap-2">
            <Bug /> オカレンス一覧
          </h1>
          
          {/* ★検索バー */}
          <div className="relative w-full md:w-96">
            <input
              type="text"
              value={searchQuery}
              onChange={handleSearch}
              placeholder="キーワードで検索 (例: タヌキ, 赤色...)"
              className="w-full p-3 pl-10 border border-gray-300 rounded-full shadow-sm focus:ring-2 focus:ring-blue-500 focus:outline-none text-black transition-all"
            />
            <Search className="absolute left-3 top-3.5 h-5 w-5 text-gray-400" />
          </div>
        </div>

        {/* ローディング表示 */}
        {loading ? (
          <div className="p-20 flex justify-center">
            <Loader2 className="animate-spin text-blue-500 h-8 w-8" />
          </div>
        ) : (
          /* リスト表示エリア */
          <div className="grid gap-4 md:grid-cols-2">
            {list.length > 0 ? (
              list.map((item) => (
                <div key={item.id} className="relative block bg-white p-6 rounded-lg shadow hover:shadow-md transition-shadow border border-gray-100 group">
                  {/* カード全体をリンクにする（絶対配置） */}
                  <Link href={`/occurrences/detail?id=${getUUID(item.id)}`} className="absolute inset-0 z-0" />

                  <h2 className="relative z-10 text-xl font-bold text-blue-700 mb-2 pointer-events-none">
                    {item.taxon_label}
                  </h2>
		  <div className="relative z-10 text-xs text-gray-500 mb-2">
			user: <span className="font-bold text-gray-700">{item.owner_name || "不明"}</span>
		  </div>
                  <p className="relative z-10 text-gray-600 text-sm line-clamp-2 pointer-events-none min-h-[1.25rem]">
                    {item.remarks || "（メモなし）"}
                  </p>
                  
                  <div className="relative z-10 mt-4 flex justify-between items-end">
                    <span className="text-xs text-gray-400 font-mono truncate max-w-[10rem]">
                        ID: {getUUID(item.id).substring(0, 8)}...
                    </span>

                    {/* ★おまけ: 種ページへのリンクボタン (z-indexを上げてクリック可能に) */}
                    {/* ※まだ種ページを作っていない場合は、このリンクは機能しないけど置いておくのだ */}
                    <Link 
                      href={`/taxon?id=ncbi:34844&name=${item.taxon_label}`}
                      className="inline-flex items-center text-xs font-bold text-purple-600 bg-purple-50 px-3 py-1 rounded-full hover:bg-purple-100 transition-colors z-20"
                    >
                      <Database className="h-3 w-3 mr-1" />
                      種データ
                    </Link>
                  </div>
                </div>
              ))
            ) : (
              <div className="col-span-2 text-center py-20 text-gray-400 bg-white rounded-lg border border-dashed border-gray-300">
                <p className="text-lg">データが見つからないのだ...</p>
                <p className="text-sm mt-2">別のキーワードで試してみてほしいのだ。</p>
              </div>
            )}
          </div>
        )}
      </div>
    </main>
  );
}
