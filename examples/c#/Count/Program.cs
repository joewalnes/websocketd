using System;
using System.Linq;
using System.Threading;

namespace Count
{
    class Program
    {
        static void Main(string[] args)
        {
            foreach (var i in Enumerable.Range(1, 10))
            {
                Console.WriteLine(i);
                Thread.Sleep(1000);
            }
        }
    }
}