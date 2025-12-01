"use client";

import { useState } from "react";
import { meiliClient, INDEX_ONTOLOGY } from "@/lib/meilisearch";
import { Search, Loader2, Plus } from "lucide-react";

// 型定義をエクスポートしておく（他のファイルでも使うから）
export type SearchResult = {
  id: string;
  label: string;
  en: string;
  uri: string;
};

// Propsの定義：親から関数をもらう
type Props = {
  onSelect: (item: SearchResult) => void;
};

export default function OntologySearch({ onSelect }: Props) {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);

  const handleSearch = async (text: string) => {
    setQuery(text);
    if (text.length === 0) {
      setResults([]);
      return;
    }

    setLoading(true);
    try {
      const search = await meiliClient.index(INDEX_ONTOLOGY).search(text, { limit: 5 });
      setResults(search.hits as SearchResult[]);
    } catch (error) {
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  const handleClick = (item: SearchResult) => {
    onSelect(item); // 親に通知！
    setQuery("");   // 入力欄をクリア
    setResults([]); // リストを消す
  };

  return (
    <div className="relative w-full">
      <div className="relative">
        <input
          type="text"
          value={query}
          onChange={(e) => handleSearch(e.target.value)}
          placeholder="形質を検索して追加 (例: 赤, red...)"
          className="w-full p-2 pl-10 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 text-black"
        />
        <div className="absolute left-3 top-2.5 text-gray-400">
          {loading ? <Loader2 className="animate-spin h-5 w-5" /> : <Search className="h-5 w-5" />}
        </div>
      </div>

      {results.length > 0 && (
        <ul className="absolute z-50 w-full bg-white border border-gray-200 rounded-md shadow-lg mt-1 max-h-60 overflow-auto">
          {results.map((item) => (
            <li
              key={item.id}
              onClick={() => handleClick(item)}
              className="p-2 hover:bg-blue-50 cursor-pointer border-b flex justify-between items-center group text-black"
            >
              <div>
                <div className="font-bold">{item.label}</div>
                <div className="text-xs text-gray-500">{item.en}</div>
              </div>
              <Plus className="h-4 w-4 text-gray-400 group-hover:text-blue-500" />
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
