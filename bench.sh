sum=0
loop=5
for((i=0;i<${loop};i++)); do
    bench=`redis-benchmark -t set | grep per | awk '{print $1}'`;
    sum=$((${sum%.*}+${bench%.*}))
done
echo $((sum/loop))