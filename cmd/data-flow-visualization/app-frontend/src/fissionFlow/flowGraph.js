import React, {useEffect, useState} from 'react'
import G6 from '@antv/g6';


export default function ({data}) {
    const ref = React.useRef(null)
    const [graph, setGraph] = useState(null)
    useEffect(() => {
        console.log(data)

        if (graph == null) {
            let tGraph = new G6.Graph({
                container: ref.current,
                width: 1600,
                height: 1200,
                fitView: true,
                modes: {
                    default: ['drag-canvas', 'zoom-canvas', 'drag-node'],
                },
                layout: {
                    type: 'dagre',
                    rankdir: 'LR',
                    align: 'DL',
                    nodesepFunc: () => 1,
                    ranksepFunc: () => 1,
                }
            });
            console.log("create")
            tGraph.data(data);
            tGraph.render();
            setGraph(tGraph)
        } else {
            console.log("update")
            graph.data(data);
            graph.render();
            // graph.paint();
        }

    }, [graph, data])

    return (
        <div ref={ref}/>
    );
}

