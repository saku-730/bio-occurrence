import { MeiliSearch } from 'meilisearch';

// 本来は環境変数(.env)にするべきだけど、今はハードコードでいくのだ
const MEILI_URL = 'http://localhost:7700';
const MEILI_KEY = 'masterKey123'; // docker-composeで決めたやつ

// シングルトン（使い回し）クライアントを作成
export const meiliClient = new MeiliSearch({
  host: MEILI_URL,
  apiKey: MEILI_KEY,
});

// インデックス名も定数にしておく
export const INDEX_ONTOLOGY = 'ontology';
