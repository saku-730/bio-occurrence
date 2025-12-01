"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Loader2, Bug } from "lucide-react";

type ListItem = {
  id: string;
  taxon_label: string;
  remarks: string;
};

export default function OccurrenceList() {
  const [list, setList] = useState<ListItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // GoのAPIから一覧を取得
    const fetchList = async () => {
      try {
        const res = await fetch("http://localhost:8080/api/occurrences");
        if (!res.ok) throw new Error("取得失敗");
        const data = await res.json();
        setList(data);
      } catch (err) {
        console.error(err);
      } finally {
        setLoading(false);
      }
    };
    fetchList();
  }, []);

  // URLからUUIDだけを抜き出すヘルパー関数
  // "http://.../occ/1234-5678" -> "1234-5678"
  const getUUID = (uri: string) => uri.split("/").pop();

  if (loading) return <div className="p-10 flex justify-center"><Loader2 className="animate-spin" /></div>;

  return (
    <main className="min-h-screen bg-gray-50 py-10 px-4">
      <div className="max-w-4xl mx-auto">
        <h1 className="text-3xl font-bold text-gray-800 mb-6 flex items-center gap-2">
           オカレンス一覧
        </h1>

        <div className="grid gap-4 md:grid-cols-2">
          {list.map((item) => (
            <Link 
              key={item.id} 
	      href={`/occurrences/detail?id=${getUUID(item.id)}`}
              className="block bg-white p-6 rounded-lg shadow hover:shadow-md transition-shadow border border-gray-100"
            >
              <h2 className="text-xl font-bold text-blue-700 mb-2">
                {item.taxon_label}
              </h2>
              <p className="text-gray-600 text-sm line-clamp-2">
                {item.remarks || "（メモなし）"}
              </p>
              <div className="mt-4 text-xs text-gray-400 font-mono truncate">
                ID: {getUUID(item.id)}
              </div>
            </Link>
          ))}
        </div>
        
        {list.length === 0 && (
          <p className="text-center text-gray-500 mt-10">まだデータがないのだ。</p>
        )}
      </div>
    </main>
  );
}
