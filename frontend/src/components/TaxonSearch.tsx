"use client";

import { useState, useEffect } from "react";
import { meiliClient, INDEX_CLASSIFICATION } from "@/lib/meilisearch";
import { Search, Loader2 } from "lucide-react";

// 検索結果の型
export type TaxonResult = {
  id: string;
  label: string;
  en: string;
  uri: string;
  ontology: string;
};

type Props = {
  initialValue?: string;
  onSelect: (item: TaxonResult | null) => void; // nullならクリア
};

export default function TaxonSearch({ initialValue, onSelect }: Props) {
  const [query, setQuery] = useState(initialValue || "");
  const [results, setResults] = useState<TaxonResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [isOpen, setIsOpen] = useState(false);

  // 親からの初期値が変わったら反映
  useEffect(() => {
    setQuery(initialValue || "");
  }, [initialValue]);

  const handleSearch = async (text: string) => {
    setQuery(text);
    setIsOpen(true);

    if (text.length < 2) { // 2文字以上で検索開始（ヒット数が多すぎるのを防ぐ）
      setResults([]);
      return;
    }

    setLoading(true);
    try {
      const search = await meiliClient.index(INDEX_CLASSIFICATION).search(text, { limit: 10 });
      setResults(search.hits as TaxonResult[]);
    } catch (error) {
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  const handleSelect = (item: TaxonResult) => {
    setQuery(item.label);
    onSelect(item);
    setIsOpen(false);
  };

  // 入力欄からフォーカスが外れた時の処理（少し遅らせないとクリック判定前に消える）
  const handleBlur = () => {
    setTimeout(() => setIsOpen(false), 200);
  };

  return (
    <div className="relative w-full">
      <div className="relative">
        <input
          type="text"
          value={query}
          onChange={(e) => {
            handleSearch(e.target.value);
            if (e.target.value === "") onSelect(null); // クリア通知
          }}
          onFocus={() => query.length >= 2 && setIsOpen(true)}
          onBlur={handleBlur}
          placeholder="生物名を検索 (例: タヌキ, Homo sapiens...)"
          className="w-full p-2 pl-10 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 text-black"
        />
        <div className="absolute left-3 top-2.5 text-gray-400">
          {loading ? <Loader2 className="animate-spin h-5 w-5" /> : <Search className="h-5 w-5" />}
        </div>
      </div>

      {isOpen && results.length > 0 && (
        <ul className="absolute z-50 w-full bg-white border border-gray-200 rounded-md shadow-lg mt-1 max-h-60 overflow-auto">
          {results.map((item) => (
            <li
              key={item.id}
              onClick={() => handleSelect(item)}
              className="p-2 hover:bg-blue-50 cursor-pointer border-b text-black"
            >
              <div className="font-bold">{item.label}</div>
              <div className="text-xs text-gray-500 flex justify-between">
                <span>{item.en}</span>
                <span className="font-mono bg-gray-100 px-1 rounded">{item.id}</span>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
