import {useState, useCallback, useEffect, useRef} from "react";
import { Input } from "@synnaxlabs/pluto/input";
import { Dropdown } from "@synnaxlabs/pluto/dropdown";
import { Text } from "@synnaxlabs/pluto/text";
import { Align } from "@synnaxlabs/pluto/align";

type SearchResult = {
    objectID: string;
    title: string;
    description: string;
    content: string;
    href: string;
}

export const Search = () => {
    const d = Dropdown.use();
    const [results, setResults] = useState<SearchResult[]>([]);
    const [value, setValue] = useState<string>("");
    const inputRef = useRef();

    const handleSearch = useCallback(async (query: string) => {
        const ALGOLIA_APP_ID = "YWD9T0JXCS"
        const ALGOLIA_SEARCH_ONLY_API_KEY = "1f8b0497301392c94adedf89a98afb6f"

        setValue(query)

        const res = await fetch(
        `https://${ALGOLIA_APP_ID}-dsn.algolia.net/1/indexes/docs_site/query`,
        {
            method: "POST",
            headers: {
            "X-Algolia-API-Key": ALGOLIA_SEARCH_ONLY_API_KEY,
            "X-Algolia-Application-Id": ALGOLIA_APP_ID,
            },
            body: JSON.stringify({
            params: `query=${query}&hitsPerPage=5&attributesToSnippet=content,title:20&highlightPreTag=<b>&highlightPostTag=</b>`,
            })
        }
        )
        const json = await res.json()
        setResults(json.hits.map((hit) => ({
            objectID: hit.objectID,
            title: hit._snippetResult?.title?.value ?? hit.title,
            description: hit.description,
            content: hit._snippetResult.content.value,
            href: hit.href,
        })))
    }, []);

    if (!d.visible && value !== "") setValue("")
    
    useEffect(() => {
        window.addEventListener("keydown", (e) => {
            if (e.key === "Escape") {
                d.close()
                inputRef.current?.blur()
            }
            if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
                e.preventDefault()
                inputRef.current?.focus()
            }
        })
    }, [ ])


    return (
        <Dropdown.Dialog {...d} className="search-box" matchTriggerWidth>
            <Input.Text ref={inputRef} size="small" placeholder="Search" value={value} onChange={handleSearch} onFocus={() => {d.open()}} 
            />
            <>
                {results.map((result) => (
                    <Align.Space el="a" direction="y" size="small" className="search-result" 
                    onClick={() => {d.close()}}
                    href={result.href} key={result.objectID}>
                        <Text.Text level="h5" dangerouslySetInnerHTML={{__html: result.title}}></Text.Text>
                        <Text.Text level="small" dangerouslySetInnerHTML={{__html: result.content}}></Text.Text>
                    </Align.Space>
                ))
                }
            </>
        </Dropdown.Dialog>
    )
}
