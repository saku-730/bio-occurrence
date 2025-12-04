"use client";

import { useState } from "react";
import { Search, Loader2, Plus, ArrowRight } from "lucide-react";
import { meiliClient, INDEX_ONTOLOGY, INDEX_DWC } from "@/lib/meilisearch";

// 検索結果の型
export type SearchResult = {
  id: string;
  label: string;
  en: string;
  uri: string;
  ontology: string; // pato, ro, envo, etc.
};

// 親コンポーネントに渡すデータの型
export type TraitItem = {
  predicateID: string;
  predicateLabel: string;
  valueID: string;
  valueLabel: string;
};

type Props = {
  onAdd: (item: TraitItem) => void;
};

export default function OntologySearch({ onAdd }: Props) {
  // 述語 (Predicate) の状態
  const [predQuery, setPredQuery] = useState("");
  const [predID, setPredID] = useState(""); // 空なら独自入力扱い
  const [predResults, setPredResults] = useState<SearchResult[]>([]);
  
  // 値 (Value) の状態
  const [valQuery, setValQuery] = useState("");
  const [valID, setValID] = useState(""); // 空なら独自入力扱い
  const [valResults, setValResults] = useState<SearchResult[]>([]);

  const [loading, setLoading] = useState(false);

  const search = async (
    indexName: string, // ★引数追加
    text: string, 
    setResults: (res: SearchResult[]) => void
  ) => {
    if (!text) {
      setResults([]);
      return;
    }
    setLoading(true);
    try {
      // 指定されたインデックスで検索！
      const res = await meiliClient.index(indexName).search(text, { limit: 10 });
      setResults(res.hits as SearchResult[]);
    } catch (error) {
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  // 述語の入力ハンドラ
  const handlePredChange = (val: string) => {
    setPredQuery(val);
    setPredID(""); // 入力を変えたらIDはリセット（独自入力状態）
    // RO (Relation Ontology) に絞って検索すると精度が良い
    search(INDEX_DWC, val, setPredResults);
  };

  // 値の入力ハンドラ
  const handleValChange = (val: string) => {
    setValQuery(val);
    setValID(""); // 入力を変えたらIDはリセット
    search(INDEX_ONTOLOGY, val, setValResults);
  };

  // 候補選択ハンドラ
  const selectPred = (item: SearchResult) => {
    setPredQuery(item.label);
    setPredID(item.id); // ID確定
    setPredResults([]);
  };

  const selectVal = (item: SearchResult) => {
    setValQuery(item.label);
    setValID(item.id); // ID確定
    setValResults([]);
  };

  // 追加ボタン
  const handleAdd = () => {
    if (!predQuery || !valQuery) return;

    onAdd({
      predicateID: predID,
      predicateLabel: predQuery,
      valueID: valID,
      valueLabel: valQuery,
    });

    // クリア
    setPredQuery("");
    setPredID("");
    setValQuery("");
    setValID("");
  };

  return (
    <div className="flex flex-col gap-2 bg-gray-50 p-4 rounded-lg border border-gray-200">
      <div className="text-xs font-bold text-gray-500 uppercase tracking-wider">新しい特徴を追加</div>
      
      <div className="flex flex-col md:flex-row gap-2 items-start md:items-center">
        
        {/* 1. 述語 (Predicate) 入力 */}
        <div className="relative flex-1 w-full">
          <div className="relative">
            <input
              type="text"
              value={predQuery}
              onChange={(e) => handlePredChange(e.target.value)}
              placeholder="項目 (例: 食性, 色...)"
              className={`w-full p-2 pl-8 border rounded text-sm ${predID ? "bg-blue-50 border-blue-300 text-blue-900" : "border-gray-300 text-black"}`}
            />
            <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-gray-400" />
          </div>
          {/* 候補リスト */}
          {predResults.length > 0 && (
            <ul className="absolute z-50 w-full bg-white border border-gray-200 rounded shadow-lg mt-1 max-h-48 overflow-auto">
              {predResults.map((item) => (
                <li key={item.id} onClick={() => selectPred(item)} className="p-2 hover:bg-gray-100 cursor-pointer text-sm border-b text-black">
                  <span className="font-bold">{item.label}</span> <span className="text-xs text-gray-500">({item.ontology})</span>
                </li>
              ))}
            </ul>
          )}
        </div>

        <ArrowRight className="hidden md:block text-gray-400 h-4 w-4" />

        {/* 2. 値 (Value) 入力 */}
        <div className="relative flex-1 w-full">
          <div className="relative">
            <input
              type="text"
              value={valQuery}
              onChange={(e) => handleValChange(e.target.value)}
              placeholder="値 (例: 昆虫, 赤...)"
              className={`w-full p-2 pl-8 border rounded text-sm ${valID ? "bg-green-50 border-green-300 text-green-900" : "border-gray-300 text-black"}`}
            />
            <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-gray-400" />
          </div>
          {/* 候補リスト */}
          {valResults.length > 0 && (
            <ul className="absolute z-50 w-full bg-white border border-gray-200 rounded shadow-lg mt-1 max-h-48 overflow-auto">
              {valResults.map((item) => (
                <li key={item.id} onClick={() => selectVal(item)} className="p-2 hover:bg-gray-100 cursor-pointer text-sm border-b text-black">
                  <span className="font-bold">{item.label}</span> <span className="text-xs text-gray-500">({item.ontology})</span>
                </li>
              ))}
            </ul>
          )}
        </div>

        {/* 3. 追加ボタン */}
        <button
          type="button"
          onClick={handleAdd}
          disabled={!predQuery || !valQuery}
          className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1 whitespace-nowrap font-bold text-sm h-9"
        >
          <Plus className="h-4 w-4" /> 追加
        </button>

      </div>
      
      <div className="text-xs text-gray-400">
        ※ 値は候補にない言葉もそのまま登録できる（独自タグになる）。
      </div>
    </div>
  );
}
