"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Loader2, Bug, Search, Database, Eye } from "lucide-react";
import { useAuth } from "@/contexts/AuthContext";

// リスト表示用のデータ型
type ListItem = {
  id: string;
  taxon_label: string;
  remarks: string;
  owner_name?: string;
  owner_id?: string;
  created_at?: string;
};

export default function OccurrenceList() {
  const [list, setList] = useState<ListItem[]>([]);
  const [loading, setLoading] = useState(true);
  
  const [searchQuery, setSearchQuery] = useState("");
  const [debounceTimer, setDebounceTimer] = useState<NodeJS.Timeout | null>(null);

  const [showMineOnly, setShowMineOnly] = useState(true);

  const { token, user } = useAuth();

  const fetchData = async (query: string) => {
    setLoading(true);
    try {
      const url = query 
        ? `http://localhost:8080/api/search?q=${encodeURIComponent(query)}`
        : "http://localhost:8080/api/occurrences";

      const headers: HeadersInit = {};
      if (token) {
        headers["Authorization"] = `Bearer ${token}`;
      }

      const res = await fetch(url, { headers });
      
      if (!res.ok) throw new Error("取得失敗");
      const data = await res.json();
      
      setList(data || []);
    } catch (err) {
      console.error(err);
      setList([]);
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (dateStr?: string) => {
   if (!dateStr) return "";
   return new Date(dateStr).toLocaleDateString("ja-JP", {
	year: "numeric",
	month: "short",
	day: "numeric",
	hour: "2-digit",
	minute: "2-digit",
    });
  };
  useEffect(() => {
    fetchData(searchQuery);
  }, [token]);

  const handleSearch = (e: React.ChangeEvent<HTMLInputElement>) => {
    const val = e.target.value;
    setSearchQuery(val);

    if (debounceTimer) clearTimeout(debounceTimer);

    const timer = setTimeout(() => {
      fetchData(val);
    }, 300);
    setDebounceTimer(timer);
  };

  const getUUID = (uri: string) => uri.split("/").pop() || "";

  const displayedList = list.filter((item) => {
    if (!user || !showMineOnly) return true;
    return item.owner_id === user.id;
  });

  return (
    <main className="min-h-screen bg-gray-50 py-8 px-4">
      <div className="max-w-7xl mx-auto">
        
        <div className="flex flex-col md:flex-row justify-between items-center mb-6 gap-4">
          <h1 className="text-2xl font-bold text-gray-800 flex items-center gap-2">
            オカレンス一覧
          </h1>
          
          <div className="flex flex-col md:flex-row gap-4 items-center w-full md:w-auto">
            {user && (
              <label className="flex items-center gap-2 text-sm text-gray-700 cursor-pointer bg-white px-3 py-2 rounded-md border border-gray-200 shadow-sm hover:bg-gray-50 transition-colors select-none whitespace-nowrap">
                <input
                  type="checkbox"
                  checked={showMineOnly}
                  onChange={(e) => setShowMineOnly(e.target.checked)}
                  className="w-4 h-4 text-blue-600 rounded focus:ring-blue-500"
                />
                自分のデータだけ表示
              </label>
            )}

            <div className="relative w-full md:w-96">
              <input
                type="text"
                value={searchQuery}
                onChange={handleSearch}
                placeholder="キーワード検索 (生物名, 特徴, ID...)"
                className="w-full p-2 pl-9 border border-gray-300 rounded-md shadow-sm focus:ring-2 focus:ring-blue-500 focus:outline-none text-sm text-black"
              />
              <Search className="absolute left-3 top-2.5 h-4 w-4 text-gray-400" />
            </div>
          </div>
        </div>

        {loading ? (
          <div className="p-20 flex justify-center">
            <Loader2 className="animate-spin text-blue-500 h-8 w-8" />
          </div>
        ) : (
          <div className="bg-white rounded-lg shadow border border-gray-200 overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-sm text-left text-gray-600">
                <thead className="text-xs text-gray-700 uppercase bg-gray-100 border-b border-gray-200">
                  <tr>
                    {/* ★変更: 操作列を左端に移動し、「詳細」に変更 */}
                    <th scope="col" className="px-6 py-3 font-bold w-24">詳細</th>
                    <th scope="col" className="px-6 py-3 font-bold w-24">ID (UUID)</th>
                    <th scope="col" className="px-6 py-3 font-bold">生物名 (Taxon)</th>
                    <th scope="col" className="px-6 py-3 font-bold">登録者</th>
		    <th scope="col" className="px-6 py-3 font-bold">登録日</th>
                    <th scope="col" className="px-6 py-3 font-bold">メモ</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {displayedList.length > 0 ? (
                    displayedList.map((item) => (
                      <tr key={item.id} className="hover:bg-blue-50 transition-colors group">
                        
                        {/* ★変更: 操作ボタンをここ（左端）に配置 */}
                        <td className="px-6 py-4 whitespace-nowrap">
                          <div className="flex gap-2">
                            <Link 
                              href={`/occurrences/detail?id=${getUUID(item.id)}`}
                              className="p-1.5 text-gray-500 hover:text-blue-600 hover:bg-blue-100 rounded-md transition-colors"
                              title="詳細を見る"
                            >
                              詳細
                            </Link>
                            
                            <Link 
                              href={`/taxon?id=ncbi:34844&name=${item.taxon_label}`}
                              className="p-1.5 text-gray-500 hover:text-purple-600 hover:bg-purple-100 rounded-md transition-colors"
                              title="種データを見る"
                            >
                              <Database className="h-4 w-4" />
                            </Link>
                          </div>
                        </td>

                        <td className="px-6 py-4 font-mono text-xs text-gray-400 whitespace-nowrap">
                           <Link href={`/occurrences/detail?id=${getUUID(item.id)}`} className="hover:text-blue-600 hover:underline">
                             {getUUID(item.id).substring(0, 8)}...
                           </Link>
                        </td>
                        
                        <td className="px-6 py-4 font-bold text-gray-900">
                          {item.taxon_label}
                        </td>

                        <td className="px-6 py-4">
                          {item.owner_name ? (
                            <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${item.owner_id === user?.id ? "bg-blue-100 text-blue-800" : "bg-gray-100 text-gray-800"}`}>
                              {item.owner_name} {item.owner_id === user?.id && "(自分)"}
                            </span>
                          ) : (
                            <span className="text-gray-300">-</span>
                          )}
                        </td>

			<td className="px-6 py-4 text-xs text-gray-500 whitespace-nowrap">
			  {formatDate(item.created_at)}
			</td>

                        <td className="px-6 py-4 max-w-xs truncate" title={item.remarks}>
                          {item.remarks || <span className="text-gray-300">-</span>}
                        </td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td colSpan={5} className="px-6 py-12 text-center text-gray-400 bg-gray-50">
                        {list.length > 0 ? "条件に合うデータが見つからないのだ..." : "データが見つからないのだ..."}
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
            
            <div className="px-6 py-3 border-t border-gray-200 bg-gray-50 text-xs text-gray-500 flex justify-between">
               <span>表示: {displayedList.length} / 全 {list.length} 件</span>
            </div>
          </div>
        )}
      </div>
    </main>
  );
}
